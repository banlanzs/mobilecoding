// mobilecoding CLI：启动 relay 连接并运行指定的 CLI 命令。
package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/gorilla/websocket"

	"github.com/banlanzs/mobilecoding/internal/relay"
)

func main() {
	relayURL := flag.String("relay", "wss://localhost:8443/relay/agent", "Relay server URL")
	insecure := flag.Bool("insecure", true, "Skip TLS certificate verification")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: mobilecoding [flags] <command> [args...]\n")
		fmt.Fprintf(os.Stderr, "Example: mobilecoding claude\n")
		fmt.Fprintf(os.Stderr, "         mobilecoding codex\n")
		os.Exit(1)
	}

	command := args[0]
	commandArgs := args[1:]

	fmt.Printf("=== MobileCoding ===\n")
	fmt.Printf("Command: %s %v\n", command, commandArgs)
	fmt.Printf("Connecting to relay: %s\n\n", *relayURL)

	// 创建 WebSocket dialer
	dialer := &websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: *insecure,
		},
	}

	// 连接到 relay 服务器
	conn, _, err := dialer.Dial(*relayURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to relay: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 发送注册帧
	regFrame := relay.AgentRegisterFrame{
		Type:      relay.TypeAgentRegister,
		Version:   relay.Version,
		SessionID: "",
	}
	if err := conn.WriteJSON(regFrame); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send register frame: %v\n", err)
		os.Exit(1)
	}

	// 读取注册响应
	_, raw, err := conn.ReadMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read register response: %v\n", err)
		os.Exit(1)
	}

	var resp relay.AgentRegisteredFrame
	if err := json.Unmarshal(raw, &resp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse register response: %v\n", err)
		os.Exit(1)
	}

	if resp.Type != relay.TypeAgentRegistered {
		fmt.Fprintf(os.Stderr, "Unexpected response type: %s\n", resp.Type)
		os.Exit(1)
	}

	sessionID := resp.SessionID
	fmt.Printf("✓ Registered with session: %s\n", sessionID)
	fmt.Printf("\n=== Scan QR Code to Connect ===\n")
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Printf("================================\n\n")
	fmt.Printf("Waiting for client to connect... (press Enter to skip)\n")

	// 等待 client 连接或用户按 Enter
	clientConnected := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		close(clientConnected)
	}()

	// 监听 relay 消息
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}

			var frame relay.ControlFrame
			if err := json.Unmarshal(raw, &frame); err != nil {
				continue
			}

			switch frame.Type {
			case relay.TypeClientAttached:
				fmt.Printf("\n✓ Client connected! Starting %s...\n\n", command)
				close(clientConnected)
			case relay.TypeRelayForward:
				// 转发 CLI 的输出到 client
			}
		}
	}()

	// 等待 client 连接或用户跳过
	<-clientConnected

	// 启动 CLI 命令
	cmd := exec.Command(command, commandArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}
}
