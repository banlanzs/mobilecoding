package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// runLocal 运行本地模式：
// 1. 启动 mobilecoding server（后台子进程）
// 2. 启动 CLI（如 claude），继承 stdin/stdout
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

	// 打印连接信息
	printConnectInfo(session)

	// 2. 启动 CLI 子进程
	cliCmd := exec.Command(session.Command, session.Args...)
	cliCmd.Stdin = os.Stdin
	cliCmd.Stdout = os.Stdout
	cliCmd.Stderr = os.Stderr

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

	// 3. 轮询 server 检测客户端连接
	clientConnected := make(chan struct{})
	go pollClientStatus(session.ServerAddr, clientConnected)

	select {
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m%s 退出: %v\033[0m\n", session.Command, err)
		}
		return ExitLoop
	case <-clientConnected:
		// 手机连接，杀掉本地 CLI，切到远程模式
		fmt.Fprintf(os.Stderr, "\n\033[33m⟳ 手机已连接，切换到远程模式...\033[0m\n")
		if cliCmd.Process != nil {
			cliCmd.Process.Signal(syscall.SIGTERM)
			<-done
		}
		return SwitchToRemote
	case sig := <-session.switchCh:
		if cliCmd.Process != nil {
			cliCmd.Process.Signal(syscall.SIGTERM)
			<-done
		}
		return sig
	}
}

// startServer 启动 mobilecoding server 子进程。
func startServer(port string) *exec.Cmd {
	// 查找 mobilecoding 可执行文件：优先 dist/，再 PATH
	serverBin := findServerBinary()
	if serverBin == "" {
		fmt.Fprintf(os.Stderr, "找不到 mobilecoding 可执行文件，请先运行 make build\n")
		return nil
	}
	cmd := exec.Command(serverBin, "-port", port)
	cmd.Stdout = os.Stderr // server 日志输出到 stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "启动服务器失败: %v\n", err)
		return nil
	}
	return cmd
}

// findServerBinary 查找 mobilecoding 可执行文件。
func findServerBinary() string {
	// 1. 尝试 dist/ 目录（开发环境）
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
	// 2. 尝试 PATH
	for _, name := range []string{"mobilecoding", "mobilecoding.exe"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// stopServer 停止 server 子进程。
func stopServer(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGTERM)
		cmd.Wait()
	}
}

// waitForServer 等待 server 就绪。
func waitForServer(addr string) bool {
	client := &http.Client{Timeout: 1 * time.Second}
	// 跳过 TLS 验证（自签名证书）
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

// pollClientStatus 轮询 server 检测是否有 WebSocket 客户端连接。
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

// printConnectInfo 打印手机连接信息。
func printConnectInfo(session *Session) {
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  ╔══════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stderr, "  ║  手机浏览器访问:                      ║\n")
	fmt.Fprintf(os.Stderr, "  ║  https://%s              ║\n", session.ServerAddr)
	fmt.Fprintf(os.Stderr, "  ╚══════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stderr, "\n")
}
