package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// runLocal 启动 server，并让 server 通过 PTY/Pipe 原生桥接 Claude CLI。
// 手机扫码后可直接操控由 server 持有的本地 CLI 进程，同时保留权限 hook。
func runLocal(session *Session) SwitchSignal {
	fmt.Fprintf(os.Stderr, "\033[32m✓ 启动 %s\033[0m\n", session.Command)

	// 1. 启动 server；server 会在 remote-control 模式下启动 native runner。
	serverCmd := startServer(session)
	if serverCmd == nil {
		return ExitLoop
	}
	defer stopServer(serverCmd)

	if !waitForServer(session.ServerAddr) {
		fmt.Fprintf(os.Stderr, "\033[31m服务器启动超时\033[0m\n")
		return ExitLoop
	}
	fmt.Fprintf(os.Stderr, "  服务器已启动: https://%s\n", session.ServerAddr)
	printConnectInfo(session)

	// 2. server 进程持有 Claude CLI。当前进程只负责转发退出信号并等待 server 退出。
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	done := make(chan error, 1)
	go func() {
		done <- serverCmd.Wait()
	}()

	select {
	case sig := <-sigCh:
		if serverCmd.Process != nil {
			_ = serverCmd.Process.Signal(sig)
		}
		if err := <-done; err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mserver 退出: %v\033[0m\n", err)
		}
	case err := <-done:
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mserver 退出: %v\033[0m\n", err)
		}
	}
	return ExitLoop
}

func startServer(session *Session) *exec.Cmd {
	serverBin := findServerBinary()
	if serverBin == "" {
		fmt.Fprintf(os.Stderr, "找不到 mobilecoding 可执行文件，请先运行 make build\n")
		return nil
	}
	args := []string{
		"-port", session.Port,
		"-launch-mode", "remote-control",
		"-default-command", session.Command,
	}
	if len(session.Args) > 0 {
		args = append(args, "-default-args", quoteArgs(session.Args))
	}
	cmd := exec.Command(serverBin, args...)
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

func quoteArgs(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			parts = append(parts, "''")
			continue
		}
		if strings.IndexFunc(arg, func(r rune) bool { return r <= ' ' || r == '\'' || r == '"' }) == -1 {
			parts = append(parts, arg)
			continue
		}
		if !strings.Contains(arg, "\"") {
			parts = append(parts, "\""+arg+"\"")
			continue
		}
		if !strings.Contains(arg, "'") {
			parts = append(parts, "'"+arg+"'")
			continue
		}
		// config.SplitArgs 没有反斜杠转义语义；同时包含单双引号时退化为空格安全表示。
		parts = append(parts, "\""+strings.ReplaceAll(arg, "\"", "")+"\"")
	}
	return strings.Join(parts, " ")
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
