package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// runLocal 运行本地模式：
// 1. 启动 mobilecoding server（后台子进程）
// 2. 启动 CLI（如 claude），通过 pipe 捕获 session_id
// 3. 轮询 server 检测手机连接，自动切换到远程模式
func runLocal(session *Session) SwitchSignal {
	fmt.Fprintf(os.Stderr, "\033[32m✓ 本地模式：直接在终端使用 %s\033[0m\n", session.Command)

	// 1. 启动 server 子进程
	serverCmd := startServer(session.Port)
	if serverCmd == nil {
		return ExitLoop
	}
	defer stopServer(serverCmd)

	// 等待 server 就绪
	if !waitForServer(session.ServerAddr) {
		fmt.Fprintf(os.Stderr, "\033[31m服务器启动超时\033[0m\n")
		return ExitLoop
	}
	fmt.Fprintf(os.Stderr, "  服务器已启动: https://%s\n", session.ServerAddr)
	printConnectInfo(session)

	// 2. 启动 CLI 子进程，pipe stdout 捕获 session_id
	cliCmd := exec.Command(session.Command, session.Args...)
	cliCmd.Stdin = os.Stdin
	cliCmd.Stderr = os.Stderr

	stdoutPipe, err := cliCmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 stdout pipe 失败: %v\n", err)
		return ExitLoop
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	if err := cliCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动 %s 失败: %v\n", session.Command, err)
		return ExitLoop
	}

	done := make(chan error, 1)
	go func() {
		done <- cliCmd.Wait()
	}()

	// 转发信号给子进程
	go func() {
		for sig := range sigCh {
			if cliCmd.Process != nil {
				cliCmd.Process.Signal(sig)
			}
		}
	}()

	// 读取 stdout，解析 session_id，同时输出到终端
	sessionIDCh := make(chan string, 1)
	go scanSessionID(stdoutPipe, sessionIDCh)

	// 3. 轮询 server 检测客户端连接
	clientConnected := make(chan struct{})
	go pollClientStatus(session.ServerAddr, clientConnected)

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m%s 退出: %v\033[0m\n", session.Command, err)
		}
		return ExitLoop
	case resumeID := <-sessionIDCh:
		session.ResumeID = resumeID
		fmt.Fprintf(os.Stderr, "\033[36m[session captured: %s]\033[0m\n", resumeID[:12])
		// 继续等待手机连接或 CLI 退出
		select {
		case err := <-done:
			if err != nil {
				fmt.Fprintf(os.Stderr, "\033[31m%s 退出: %v\033[0m\n", session.Command, err)
			}
			return ExitLoop
		case <-clientConnected:
			return switchToRemote(session, cliCmd, done)
		case sig := <-session.switchCh:
			return sig
		}
	case <-clientConnected:
		// 手机连接但还没捕获到 session_id
		return switchToRemote(session, cliCmd, done)
	case sig := <-session.switchCh:
		if cliCmd.Process != nil {
			cliCmd.Process.Signal(syscall.SIGTERM)
			<-done
		}
		return sig
	}
}

// switchToRemote 切换到远程模式：杀掉本地 CLI，通知 server resume 会话。
func switchToRemote(session *Session, cliCmd *exec.Cmd, done chan error) SwitchSignal {
	fmt.Fprintf(os.Stderr, "\n\033[33m⟳ 手机已连接，切换到远程模式...\033[0m\n")
	if cliCmd.Process != nil {
		cliCmd.Process.Signal(syscall.SIGTERM)
		<-done
	}
	// 通知 server resume 同一个 Claude 会话
	if session.ResumeID != "" {
		if err := notifyResume(session.ServerAddr, session.ResumeID); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m通知 resume 失败: %v\033[0m\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "  会话已恢复: %s\n", session.ResumeID[:12])
		}
	}
	return SwitchToRemote
}

// scanSessionID 从 Claude 的 JSONL stdout 中提取 session_id。
// 同时将所有输出转发到 stderr（保持终端显示）。
func scanSessionID(r io.Reader, found chan<- string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		// 输出到终端
		os.Stdout.Write(line)
		os.Stdout.Write([]byte("\n"))
		// 解析 session_id
		var m map[string]any
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}
		if sid, ok := m["session_id"].(string); ok && sid != "" {
			select {
			case found <- sid:
			default:
			}
		}
	}
}

// notifyResume 通知 server 使用指定的 session_id resume Claude 会话。
func notifyResume(addr, resumeID string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = transport

	body := strings.NewReader(fmt.Sprintf(`{"resumeSessionId":"%s"}`, resumeID))
	resp, err := client.Post(
		fmt.Sprintf("https://%s/api/v1/resume", addr),
		"application/json",
		body,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}
	return nil
}

// startServer 启动 mobilecoding server 子进程。
func startServer(port string) *exec.Cmd {
	serverBin := findServerBinary()
	if serverBin == "" {
		fmt.Fprintf(os.Stderr, "找不到 mobilecoding 可执行文件，请先运行 make build\n")
		return nil
	}
	cmd := exec.Command(serverBin, "-port", port)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
		return nil
	}
	return cmd
}

func stopServer(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGTERM)
		cmd.Wait()
	}
}

func waitForServer(addr string) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = transport
	for i := 0; i < 30; i++ {
		resp, err := client.Get(fmt.Sprintf("https://%s/healthz", addr))
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}

func pollClientStatus(addr string, connected chan<- struct{}) {
	client := &http.Client{Timeout: 2 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = transport

	wasConnected := false
	for {
		time.Sleep(1 * time.Second)
		resp, err := client.Get(fmt.Sprintf("https://%s/api/v1/clients", addr))
		if err != nil {
			continue
		}
		var status struct {
			Subscribers int `json:"subscribers"`
		}
		json.NewDecoder(resp.Body).Decode(&status)
		resp.Body.Close()

		isConnected := status.Subscribers > 0
		if isConnected && !wasConnected {
			close(connected)
			return
		}
		wasConnected = isConnected
	}
}

func findServerBinary() string {
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "mobilecoding"),
			filepath.Join(dir, "mobilecoding.exe"),
			filepath.Join(dir, "..", "dist", "mobilecoding"),
			filepath.Join(dir, "..", "dist", "mobilecoding.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	for _, name := range []string{"mobilecoding", "mobilecoding.exe"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func printConnectInfo(session *Session) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  ╔══════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "  ║  手机浏览器访问:                      ║\n")
	fmt.Fprintf(os.Stderr, "  ║  https://%s              ║\n", session.ServerAddr)
	fmt.Fprintf(os.Stderr, "  ╚══════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\n")
}
