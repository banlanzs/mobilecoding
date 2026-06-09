// Package projection 把 engine.Runner 输出的原始字节流翻译成结构化事件。
package projection

import (
	"time"

	"github.com/google/uuid"
	"github.com/banlanzs/mobilecoding/internal/protocol"
)

// EventType 区分投影后的事件类型。值定义在 protocol 包中。
type EventType = string

// 向后兼容别名：引用 protocol 包常量。
const (
	EventText          = protocol.EvtText
	EventTextDelta     = protocol.EvtTextDelta
	EventLifecycle     = protocol.EvtLifecycle
	EventToolUse       = protocol.EvtToolUse
	EventToolResult    = protocol.EvtToolResult
	EventPermissionReq = protocol.EvtPermissionReq
	EventPlanMode      = protocol.EvtPlanMode
	EventContextWindow = protocol.EvtContextWindow
	EventSession       = protocol.EvtSession
	EventThinkingStart = protocol.EvtThinkingStart
	EventThinkingEnd   = protocol.EvtThinkingEnd
	EventToolStart     = protocol.EvtToolStart
	EventToolOutput    = protocol.EvtToolOutput
	EventToolEnd       = protocol.EvtToolEnd
	EventBashStart     = protocol.EvtBashStart
	EventBashOutput    = protocol.EvtBashOutput
	EventBashEnd       = protocol.EvtBashEnd
	EventAgentState    = protocol.EvtAgentState
	EventTurnEnd       = protocol.EvtTurnEnd
	EventPermissionAsk = protocol.EvtPermissionAsk
)

// Event 是投影后的事件（前端订阅的契约）。
type Event struct {
	Type       EventType `json:"type"`
	SessionID  string    `json:"sessionId"`
	Time       time.Time `json:"time"`
	Seq        int64     `json:"seq,omitempty"` // 消息序列号，由 store 分配
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

func TurnEndEvent(sid, result string, isError bool) Event {
	return Event{Type: EventTurnEnd, SessionID: sid, Time: time.Now().UTC(), Text: result, Message: result, MessageID: newMessageID()}
}

func PermissionAskEvent(sid, requestID, toolName, prompt string) Event {
	return Event{Type: EventPermissionAsk, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, Message: prompt, MessageID: requestID}
}

// PermissionAskEventWithID 是 PermissionAskEvent 的兼容版本，允许 ToolInput 作为 JSON 透传给前端。
// 用于 Claude HTTP hook 场景：toolInput 已经是 json.RawMessage。
func PermissionAskEventWithID(sid, requestID, toolName, prompt string) Event {
	return Event{Type: EventPermissionAsk, SessionID: sid, Time: time.Now().UTC(), ToolName: toolName, Message: prompt, MessageID: requestID}
}
