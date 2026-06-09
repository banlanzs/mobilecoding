// Package protocol 定义 mobilecoding WebSocket 协议常量。
// 这是前后端的唯一协议契约，所有 RPC 方法名、事件类型、错误码在此集中定义。
package protocol

// ─── Envelope 类型 ────────────────────────────────────────────────────────────

const (
	EnvTypeReq  = "req"  // 客户端 → 服务端请求
	EnvTypeResp = "resp" // 服务端 → 客户端响应
	EnvTypeEvt  = "evt"  // 服务端 → 客户端事件
)

// ─── RPC 方法名 ───────────────────────────────────────────────────────────────

const (
	MethodSessionStart            = "session.start"
	MethodSessionInput            = "session.input"
	MethodSessionStop             = "session.stop"
	MethodSessionAbort            = "session.abort"
	MethodSessionPermissionAnswer = "session.permission.answer" // 旧 stdio 权限协议
	MethodPermissionRespond       = "permission.respond"        // 新 HTTP hook 权限协议
)

// ─── RPC 错误码 ───────────────────────────────────────────────────────────────

const (
	ErrProtocolError = "protocol_error" // 参数格式错误
	ErrNotFound      = "not_found"      // 未知方法
	ErrEngineFailure = "engine_failure" // 引擎/会话错误
	ErrConflict      = "conflict"       // 会话已存在
	ErrNotConfigured = "not_configured" // 功能未配置
	ErrStaleRequest  = "stale_request"  // 请求已过期
)

// ─── 投影事件类型 ─────────────────────────────────────────────────────────────

const (
	EvtText          = "text"
	EvtTextDelta     = "text_delta"
	EvtLifecycle     = "lifecycle"
	EvtToolUse       = "tool_use"
	EvtToolResult    = "tool_result"
	EvtPermissionReq = "permission_request"
	EvtPermissionAsk = "permission_ask"
	EvtPlanMode      = "plan_mode"
	EvtContextWindow = "context_window"
	EvtSession       = "session"
	EvtThinkingStart = "thinking_start"
	EvtThinkingEnd   = "thinking_end"
	EvtToolStart     = "tool_start"
	EvtToolOutput    = "tool_output"
	EvtToolEnd       = "tool_end"
	EvtBashStart     = "bash_start"
	EvtBashOutput    = "bash_output"
	EvtBashEnd       = "bash_end"
	EvtAgentState    = "agent_state"
	EvtTurnEnd       = "turn_end"
)
