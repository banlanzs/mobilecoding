// Package projection 把 engine.Runner 输出的原始字节流翻译成结构化事件。
// MVP 1 阶段：只做 raw 文本事件与 lifecycle 透传。
// 后续 MVP：加入 diff / permission / plan / context_window 等高级投影。
package projection

import "time"

// EventType 区分投影后的事件类型。
type EventType string

const (
	EventText           EventType = "text"
	EventLifecycle      EventType = "lifecycle"
	EventToolUse        EventType = "tool_use"
	EventToolResult     EventType = "tool_result"
	EventPermissionReq  EventType = "permission_request"
	EventPlanMode       EventType = "plan_mode"
	EventContextWindow  EventType = "context_window"
	EventSession        EventType = "session"
)

// Event 是投影后的事件（前端订阅的契约）。
type Event struct {
	Type      EventType
	SessionID string
	Time      time.Time
	Text      string // EventText
	Message   string // EventLifecycle
	// 扩展字段
	ToolName  string
	ToolInput any
	ToolResult any
}

// TextEvent 构造一个文本事件。
func TextEvent(sid, text string) Event {
	return Event{Type: EventText, SessionID: sid, Time: time.Now().UTC(), Text: text}
}

// LifecycleEvent 构造一个生命周期事件。
func LifecycleEvent(sid, message string) Event {
	return Event{Type: EventLifecycle, SessionID: sid, Time: time.Now().UTC(), Message: message}
}

// ToolUseEvent 构造一个工具使用事件。
func ToolUseEvent(sid, toolName string, input any) Event {
	return Event{Type: EventToolUse, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, ToolInput: input}
}

// ToolResultEvent 构造一个工具结果事件。
func ToolResultEvent(sid, toolName string, result any) Event {
	return Event{Type: EventToolResult, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, ToolResult: result}
}

// PermissionRequestEvent 构造一个权限请求事件。
func PermissionRequestEvent(sid, toolName, prompt string) Event {
	return Event{Type: EventPermissionReq, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, Message: prompt}
}

// PlanModeEvent 构造一个计划模式事件。
func PlanModeEvent(sid string, data any) Event {
	return Event{Type: EventPlanMode, SessionID: sid, Time: time.Now().UTC(), ToolInput: data}
}

// ContextWindowEvent 构造一个上下文窗口事件。
func ContextWindowEvent(sid string, data any) Event {
	return Event{Type: EventContextWindow, SessionID: sid, Time: time.Now().UTC(), ToolInput: data}
}

// SessionEvent 构造一个会话事件。
func SessionEvent(sid string, data any) Event {
	return Event{Type: EventSession, SessionID: sid, Time: time.Now().UTC(), ToolInput: data}
}
