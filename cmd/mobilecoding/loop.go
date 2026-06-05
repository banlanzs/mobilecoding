package main

// Session 跨模式共享的会话状态。
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
