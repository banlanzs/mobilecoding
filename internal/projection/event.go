// Package projection 把 engine.Runner 输出的原始字节流翻译成结构化事件。
// MVP 1 阶段：只做 raw 文本事件与 lifecycle 透传。
// 后续 MVP：加入 diff / permission / plan / context_window 等高级投影。
package projection

import (
	"time"

	"github.com/google/uuid"
)

// EventType 区分投影后的事件类型。
type EventType string

const (
	// 现有类型（保持向后兼容）
	EventText          EventType = "text"
	EventTextDelta     EventType = "text_delta"
	EventLifecycle     EventType = "lifecycle"
	EventToolUse       EventType = "tool_use"
	EventToolResult    EventType = "tool_result"
	EventPermissionReq EventType = "permission_request"
	EventPlanMode      EventType = "plan_mode"
	EventContextWindow EventType = "context_window"
	EventSession       EventType = "session"

	// 新增统一 Agent 事件类型
	EventThinkingStart EventType = "thinking_start" // 思考开始
	EventThinkingDelta EventType = "thinking_delta" // 思考增量
	EventThinkingEnd   EventType = "thinking_end"   // 思考结束
	EventToolStart     EventType = "tool_start"     // 工具开始执行
	EventToolOutput    EventType = "tool_output"    // 工具流式输出
	EventToolEnd       EventType = "tool_end"       // 工具执行完成
	EventBashStart     EventType = "bash_start"     // Bash 命令开始
	EventBashOutput    EventType = "bash_output"    // Bash 流式输出
	EventBashEnd       EventType = "bash_end"       // Bash 命令完成
	EventAgentState    EventType = "agent_state"    // Agent 状态变更
)

// Event 是投影后的事件（前端订阅的契约）。
type Event struct {
	Type       EventType `json:"type"`
	SessionID  string    `json:"sessionId"`
	Time       time.Time `json:"time"`
	Text       string    `json:"text,omitempty"`
	Thinking   string    `json:"thinking,omitempty"` // 模型思考过程，折叠展示
	Message    string    `json:"message,omitempty"`
	// 扩展字段
	ToolName   string `json:"toolName,omitempty"`
	ToolInput  any    `json:"toolInput,omitempty"`
	ToolResult any    `json:"toolResult,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"` // 流式工具输出
	ToolID     string `json:"toolId,omitempty"`     // 工具调用唯一 ID
	BlockIndex int    `json:"blockIndex,omitempty"` // text_delta 所属的文本块序号
	MessageID  string `json:"messageId,omitempty"`  // 稳定消息标识符
	State      string `json:"state,omitempty"`      // agent_state 的状态值
}

func newMessageID() string { return uuid.NewString() }

// --- 现有构造器（保持不变）---

func TextEvent(sid, text string) Event {
	return Event{Type: EventText, SessionID: sid, Time: time.Now().UTC(), Text: text, MessageID: newMessageID()}
}

func TextEventWithThinking(sid, text, thinking string) Event {
	return Event{Type: EventText, SessionID: sid, Time: time.Now().UTC(), Text: text, Thinking: thinking, MessageID: newMessageID()}
}

func TextDeltaEvent(sid, text string, blockIndex int) Event {
	return Event{Type: EventTextDelta, SessionID: sid, Time: time.Now().UTC(), Text: text, BlockIndex: blockIndex, MessageID: newMessageID()}
}

func LifecycleEvent(sid, message string) Event {
	return Event{Type: EventLifecycle, SessionID: sid, Time: time.Now().UTC(), Message: message, MessageID: newMessageID()}
}

func ToolUseEvent(sid, toolName string, input any) Event {
	return Event{Type: EventToolUse, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, ToolInput: input, MessageID: newMessageID()}
}

func ToolResultEvent(sid, toolName string, result any) Event {
	return Event{Type: EventToolResult, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, ToolResult: result, MessageID: newMessageID()}
}

func PermissionRequestEvent(sid, toolName, prompt string) Event {
	return Event{Type: EventPermissionReq, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, Message: prompt, MessageID: newMessageID()}
}

func PlanModeEvent(sid string, data any) Event {
	return Event{Type: EventPlanMode, SessionID: sid, Time: time.Now().UTC(), ToolInput: data, MessageID: newMessageID()}
}

func ContextWindowEvent(sid string, data any) Event {
	return Event{Type: EventContextWindow, SessionID: sid, Time: time.Now().UTC(), ToolInput: data, MessageID: newMessageID()}
}

func SessionEvent(sid string, data any) Event {
	return Event{Type: EventSession, SessionID: sid, Time: time.Now().UTC(), ToolInput: data, MessageID: newMessageID()}
}

// --- 新增统一 Agent 事件构造器 ---

func ThinkingStartEvent(sid string) Event {
	return Event{Type: EventThinkingStart, SessionID: sid, Time: time.Now().UTC(), MessageID: newMessageID()}
}

func ThinkingEndEvent(sid string) Event {
	return Event{Type: EventThinkingEnd, SessionID: sid, Time: time.Now().UTC(), MessageID: newMessageID()}
}

func ToolStartEvent(sid, toolID, toolName string, input any) Event {
	return Event{Type: EventToolStart, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolName: toolName, ToolInput: input, MessageID: newMessageID()}
}

func ToolOutputEvent(sid, toolID, output string) Event {
	return Event{Type: EventToolOutput, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolOutput: output, MessageID: newMessageID()}
}

func ToolEndEvent(sid, toolID, toolName string) Event {
	return Event{Type: EventToolEnd, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolName: toolName, MessageID: newMessageID()}
}

func BashStartEvent(sid, toolID, command string) Event {
	return Event{Type: EventBashStart, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolName: "Bash", ToolInput: command, MessageID: newMessageID()}
}

func BashOutputEvent(sid, toolID, output string) Event {
	return Event{Type: EventBashOutput, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolOutput: output, MessageID: newMessageID()}
}

func BashEndEvent(sid, toolID string) Event {
	return Event{Type: EventBashEnd, SessionID: sid, Time: time.Now().UTC(), ToolID: toolID, ToolName: "Bash", MessageID: newMessageID()}
}

func AgentStateEvent(sid, state string) Event {
	return Event{Type: EventAgentState, SessionID: sid, Time: time.Now().UTC(), State: state, MessageID: newMessageID()}
}
