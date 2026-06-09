// Session 跨子命令共享的会话状态。
// 用于 claude/codex 等 mc 模式子命令传递配置。
package main

// Session 描述一个 mc 模式会话：要运行的本地 CLI 进程，以及如何连接到 server。
type Session struct {
	Command    string   // 要运行的命令（claude/codex/...）
	Args       []string // 命令参数
	Port       string   // 服务器端口
	ServerAddr string   // 服务器地址（host:port）
}

// SwitchSignal 返回值，表示 runLocal 的退出原因。
type SwitchSignal int

const (
	ExitLoop SwitchSignal = iota
)
