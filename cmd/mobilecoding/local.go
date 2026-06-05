package main

import (
	"bufio"
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

	"github.com/gorilla/websocket"
)

// runLocal 启动 server，然后 mc 作为 WebSocket 客户端连接 server。
// 终端输入通过 WebSocket 发送给 server 管理的 Claude 会话。
// 手机同时连接同一个 session，实现遥控器模式。
func runLocal(session *Session) SwitchSignal {
	fmt.Fprintf(os.Stderr, "\033[32m✓ 本地模式：启动 %s\033[0m\n", session.Command)

	// 1. 启动 server
	serverCmd := startServer(session.Port)
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

	// 2. 通过 WebSocket 连接 server
	conn, err := connectWS(session.ServerAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebSocket 连接失败: %v\n", err)
		return ExitLoop
	}
	defer conn.Close()

	// 3. 启动 Claude 会话（通过 server 管理）
	if err := startSession(conn, session.Command, session.Args); err != nil {
		fmt.Fprintf(os.Stderr, "启动会话失败: %v\n", err)
		return ExitLoop
	}

	// 4. 显示就绪提示
	fmt.Fprintf(os.Stderr, "\n\033[36m╭─ mobilecoding ─────────────────────────╮\033[0m\n")
	fmt.Fprintf(os.Stderr, "\033[36m│\033[0m 输入消息后按 Enter 发送，Ctrl+C 退出    \033[36m│\033[0m\n")
	fmt.Fprintf(os.Stderr, "\033[36m╰─────────────────────────────────────────╯\033[0m\n\n")
	fmt.Fprintf(os.Stderr, "\033[32m❯ \033[0m")

	// 5. 接收 server 事件并显示
	eventsDone := make(chan struct{})
	go receiveEvents(conn, eventsDone)

	// 6. 读取终端输入，发送给 server
	inputDone := make(chan struct{})
	go forwardInput(conn, inputDone)

	// 6. 等待退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-eventsDone:
		fmt.Fprintf(os.Stderr, "\n\033[31m会话结束\033[0m\n")
	case <-inputDone:
		// stdin 关闭（Ctrl+D）
	case sig := <-sigCh:
		fmt.Fprintf(os.Stderr, "\n收到信号 %v\n", sig)
	}

	return ExitLoop
}

// connectWS 连接 server 的 WebSocket 端点。
func connectWS(addr string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	url := fmt.Sprintf("wss://%s/api/v1/ws", addr)
	conn, _, err := dialer.Dial(url, nil)
	return conn, err
}

// startSession 通过 WebSocket 启动 Claude 会话。
func startSession(conn *websocket.Conn, command string, args []string) error {
	params := map[string]any{
		"command": command,
		"args":    args,
	}
	return sendRPC(conn, "session.start", params)
}

// forwardInput 从 stdin 读取输入，通过 WebSocket 发送给 server。
func forwardInput(conn *websocket.Conn, done chan<- struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			fmt.Fprintf(os.Stderr, "\033[32m❯ \033[0m")
			continue
		}
		if err := sendRPC(conn, "session.input", map[string]any{"text": text}); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31m发送失败: %v\033[0m\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "\033[90m⏳ 等待响应...\033[0m\n")
	}
}

// receiveEvents 从 WebSocket 接收事件并显示在终端。
func receiveEvents(conn *websocket.Conn, done chan<- struct{}) {
	defer close(done)
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var env struct {
			Type  string          `json:"type"`
			Event json.RawMessage `json:"event"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			continue
		}
		if env.Type == "evt" {
			handleEvent(env.Event)
		}
		// resp 类型忽略（RPC 响应）
	}
}

// handleEvent 处理 server 推送的事件，显示在终端。
func handleEvent(raw json.RawMessage) {
	var ev struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Thinking string `json:"thinking"`
		Message  string `json:"message"`
		ToolName string `json:"toolName"`
	}
	json.Unmarshal(raw, &ev)

	switch ev.Type {
	case "text":
		if ev.Thinking != "" {
			fmt.Fprintf(os.Stderr, "\033[90m[thinking] %s\033[0m\n", ev.Thinking)
		}
		if ev.Text != "" {
			fmt.Printf("%s", ev.Text)
		}
	case "text_delta":
		// 增量文本直接输出（不换行）
		var delta struct {
			Text     string `json:"text"`
			Thinking string `json:"thinking"`
		}
		json.Unmarshal(raw, &delta)
		if delta.Text != "" {
			fmt.Printf("%s", delta.Text)
		}
	case "tool_start":
		fmt.Fprintf(os.Stderr, "\033[36m● %s\033[0m ", ev.ToolName)
	case "tool_end":
		fmt.Fprintf(os.Stderr, "\n")
	case "bash_start":
		fmt.Fprintf(os.Stderr, "\033[33m❯ %s\033[0m\n", ev.ToolName)
	case "turn_end":
		fmt.Printf("\n\n")
		fmt.Fprintf(os.Stderr, "\033[32m❯ \033[0m")
	case "permission_request", "permission_ask":
		fmt.Fprintf(os.Stderr, "\n\033[33m⚠ 权限请求: %s — %s\033[0m\n", ev.ToolName, ev.Message)
	case "lifecycle":
		// 忽略内部生命周期事件
	}
}

// sendRPC 发送 RPC 请求到 WebSocket。
func sendRPC(conn *websocket.Conn, method string, params any) error {
	id := fmt.Sprintf("mc-%d", time.Now().UnixNano())
	envelope := map[string]any{
		"type":   "req",
		"id":     id,
		"method": method,
		"params": params,
	}
	return conn.WriteJSON(envelope)
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
