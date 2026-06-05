// Local/Remote 模式切换状态机。
// 参考 happy 的 loop.ts：while(true) + switch(mode) 极简设计。
package main

import (
	"fmt"
	"os"
)

// Mode 表示当前运行模式。
type Mode string

const (
	ModeLocal  Mode = "local"  // 终端本地直接运行 Claude
	ModeRemote Mode = "remote" // 手机通过 WebSocket 远程控制
)

// SwitchSignal 模式切换信号。
type SwitchSignal string

const (
	SwitchToLocal  SwitchSignal = "local"  // 切回本地
	SwitchToRemote SwitchSignal = "remote" // 切到远程
	ExitLoop       SwitchSignal = "exit"   // 退出循环
)

// Session 跨模式共享的会话状态。
type Session struct {
	Command    string   // 要运行的命令（claude/codex/...）
	Args       []string // 命令参数
	Port       string   // 服务器端口
	ServerAddr string   // 服务器地址（host:port）
	AuthToken  string   // 认证 token
	switchCh   chan SwitchSignal
}

// NewSession 创建新的会话实例。
func NewSession(command string, args []string, serverAddr, authToken string) *Session {
	return &Session{
		Command:    command,
		Args:       args,
		Port:       "8443",
		ServerAddr: serverAddr,
		AuthToken:  authToken,
		switchCh:   make(chan SwitchSignal, 1),
	}
}

// RequestSwitch 请求模式切换（非阻塞）。
func (s *Session) RequestSwitch(sig SwitchSignal) {
	select {
	case s.switchCh <- sig:
	default:
		// 已有待处理信号，忽略
	}
}

// WaitForSwitch 等待模式切换信号。
func (s *Session) WaitForSwitch() SwitchSignal {
	return <-s.switchCh
}

// Run 运行模式切换循环。
func Run(session *Session, startMode Mode) error {
	mode := startMode
	for {
		fmt.Fprintf(os.Stderr, "\033[36m[mode: %s]\033[0m\n", mode)
		switch mode {
		case ModeLocal:
			result := runLocal(session)
			switch result {
			case SwitchToRemote:
				mode = ModeRemote
			case ExitLoop:
				return nil
			}
		case ModeRemote:
			result := runRemote(session)
			switch result {
			case SwitchToLocal:
				mode = ModeLocal
			case ExitLoop:
				return nil
			}
		default:
			return fmt.Errorf("unknown mode: %s", mode)
		}
	}
}
