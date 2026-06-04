// Package engine 定义 AI 引擎抽象：Runner interface + 通用 PTY 实现 + 注册表。
// 不同 AI CLI（claude / codex / 任意 LLM CLI）通过不同 Runner 实现接入。
package engine

import (
	"context"
	"encoding/json"
)

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

	// SendToStdin 写入运行中进程的 stdin，不杀进程。
	// 用于权限应答等中间交互场景。
	SendToStdin(p []byte) error

	// Abort 中止当前正在执行的请求（杀进程），保留 session 等待下一条消息。
	Abort()

	InteractiveStateProvider
	TurnStateProvider
}

// PermissionAnswer 构造 Claude stream-json 权限应答。
// 同时支持 stdio control_response 协议和旧版 permission_answer 协议（兼容老版本 Claude CLI）。
func PermissionAnswer(allow bool, toolName string) []byte {
	b, _ := json.Marshal(map[string]any{
		"type":      "permission_answer",
		"allow":     allow,
		"tool_name": toolName,
	})
	return append(b, '\n')
}

// ControlResponse 构造 Claude stdio control_response 协议帧。
// 配合 control_request 事件使用：把 request_id 原样回填。
func ControlResponse(requestID string, allow bool) []byte {
	payload := map[string]any{
		"type": "control_response",
		"response": map[string]any{
			"request_id": requestID,
			"allow":      allow,
		},
	}
	b, _ := json.Marshal(payload)
	return append(b, '\n')
}
