// Package engine 定义 AI 引擎抽象：Runner interface + 通用 PTY 实现 + 注册表。
// 不同 AI CLI（claude / codex / 任意 LLM CLI）通过不同 Runner 实现接入。
package engine

import "context"

type EventKind string

const (
	EventRaw       EventKind = "raw"
	EventLifecycle EventKind = "lifecycle"
)

type Event struct {
	Kind    EventKind
	Data    []byte
	Message string
}

type ExecRequest struct {
	Command         string
	Args            []string
	CWD             string
	Env             []string
	Cols            int
	Rows            int
	VisibleTerminal bool // 是否在可视化终端窗口中启动（Windows）
}

type InteractiveStateProvider interface {
	CanAcceptInteractiveInput() bool
}

type TurnStateProvider interface {
	HasActiveTurn() bool
}

type Runner interface {
	Start(ctx context.Context, req ExecRequest) error
	Write(p []byte) error
	Resize(cols, rows int) error
	Close() error

	Events() <-chan Event
	Errors() <-chan error
	Done() <-chan struct{}

	SessionID() string

	InteractiveStateProvider
	TurnStateProvider
}
