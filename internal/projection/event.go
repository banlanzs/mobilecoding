// Package projection 把 engine.Runner 输出的原始字节流翻译成结构化事件。
// MVP 1 阶段：只做 raw 文本事件与 lifecycle 透传。
// 后续 MVP：加入 diff / permission / plan / context_window 等高级投影。
package projection

import "time"

// EventType 区分投影后的事件类型。
type EventType string

const (
	EventText      EventType = "text"
	EventLifecycle EventType = "lifecycle"
	// 后续扩展：EventDiff / EventPermission / EventPlan / EventContextWindow / EventError
)

// Event 是投影后的事件（前端订阅的契约）。
type Event struct {
	Type      EventType
	SessionID string
	Time      time.Time
	Text      string // EventText
	Message   string // EventLifecycle
}

// TextEvent 构造一个文本事件。
func TextEvent(sid, text string) Event {
	return Event{Type: EventText, SessionID: sid, Time: time.Now().UTC(), Text: text}
}

// LifecycleEvent 构造一个生命周期事件。
func LifecycleEvent(sid, message string) Event {
	return Event{Type: EventLifecycle, SessionID: sid, Time: time.Now().UTC(), Message: message}
}
