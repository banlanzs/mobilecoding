package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// runRemote 运行远程模式：手机通过 WebSocket 控制 Claude。
// CLI 等待用户按键或手机断开来触发切换。
func runRemote(session *Session) SwitchSignal {
	fmt.Fprintf(os.Stderr, "\033[36m✓ 远程模式：手机已接管控制\033[0m\n")
	fmt.Fprintf(os.Stderr, "  在此终端按 Enter 切回本地模式\n\n")

	// 监听用户按 Enter
	userSwitch := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		close(userSwitch)
	}()

	// 监听手机断开
	clientDisconnected := make(chan struct{})
	go pollClientDisconnect(session.ServerAddr, clientDisconnected)

	select {
	case <-userSwitch:
		fmt.Fprintf(os.Stderr, "\n\033[33m⟳ 切回本地模式...\033[0m\n")
		return SwitchToLocal
	case <-clientDisconnected:
		fmt.Fprintf(os.Stderr, "\n\033[32m⟳ 手机已断开，切回本地模式\033[0m\n")
		return SwitchToLocal
	case sig := <-session.switchCh:
		return sig
	}
}

// pollClientDisconnect 轮询 server 检测客户端是否断开。
func pollClientDisconnect(addr string, disconnected chan<- struct{}) {
	client := &http.Client{Timeout: 2 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client.Transport = transport

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

		if status.Subscribers == 0 {
			close(disconnected)
			return
		}
	}
}
