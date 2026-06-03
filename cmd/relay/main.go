// mobilecoding relay CLI：连接到 relay 服务器，转发终端 I/O 到手机。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"

	"github.com/banlanzs/mobilecoding/internal/relay"
)

func main() {
	relayURL := flag.String("relay", "ws://localhost:8443/relay/agent", "Relay server URL")
	sessionID := flag.String("session", "", "Session ID (optional, auto-generated if empty)")
	flag.Parse()

	fmt.Printf("=== MobileCoding Relay CLI ===\n")
	fmt.Printf("Connecting to relay: %s\n", *relayURL)

	// 连接到 relay 服务器
	conn, _, err := websocket.DefaultDialer.Dial(*relayURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to relay: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 发送注册帧
	regFrame := relay.AgentRegisterFrame{
		Type:      relay.TypeAgentRegister,
		Version:   relay.Version,
		SessionID: *sessionID,
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

	fmt.Printf("Registered with session: %s\n", resp.SessionID)
	fmt.Printf("Waiting for client to connect...\n\n")

	// 启动信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动读取协程
	go func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nRelay connection closed: %v\n", err)
				os.Exit(0)
			}

			var frame relay.ControlFrame
			if err := json.Unmarshal(raw, &frame); err != nil {
				continue
			}

			switch frame.Type {
			case relay.TypeClientAttached:
				var attachFrame relay.ClientAttachedFrame
				if err := json.Unmarshal(raw, &attachFrame); err == nil {
					fmt.Printf("\n[CLIENT CONNECTED] Client ID: %s\n", attachFrame.ClientID)
					fmt.Printf("You can now type messages. Press Ctrl+C to exit.\n\n")
				}
			case relay.TypeRelayForward:
				var env relay.ForwardEnvelope
				if err := json.Unmarshal(raw, &env); err == nil {
					// 解析 payload
					var msg map[string]interface{}
					if err := json.Unmarshal([]byte(env.Payload), &msg); err == nil {
						if text, ok := msg["text"].(string); ok {
							fmt.Printf("[CLIENT] %s\n", text)
						}
					}
				}
			case relay.TypeRelayError:
				var errFrame relay.ErrorFrame
				if err := json.Unmarshal(raw, &errFrame); err == nil {
					fmt.Fprintf(os.Stderr, "[ERROR] %s: %s\n", errFrame.Code, errFrame.Message)
				}
			}
		}
	}()

	// 启动输入协程
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			if text == "" {
				continue
			}

			// 发送消息到 relay
			payload, _ := json.Marshal(map[string]string{"text": text})
			env := relay.ForwardEnvelope{
				Type:        relay.TypeRelayForward,
				Version:     relay.Version,
				SessionID:   resp.SessionID,
				Direction:   relay.DirectionAgentToClient,
				MessageID:   fmt.Sprintf("msg_%d", os.Getpid()),
				ContentType: relay.ContentTypeMobileCoding,
				Payload:     string(payload),
			}
			conn.WriteJSON(env)
		}
	}()

	// 等待退出信号
	<-sigCh
	fmt.Printf("\nDisconnecting...\n")
}
