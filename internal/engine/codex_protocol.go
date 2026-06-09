package engine

import "encoding/json"

// Codex JSON-RPC 协议常量（参考 easycodex codex-rpc.ts）。
const (
	// 客户端 → 服务端
	CodexMethodInitialize = "initialize"
	CodexMethodThreadStart  = "thread/start"
	CodexMethodThreadResume = "thread/resume"
	CodexMethodThreadList   = "thread/list"
	CodexMethodThreadRead   = "thread/read"
	CodexMethodTurnStart    = "turn/start"
	CodexMethodTurnInterrupt = "turn/interrupt"
	CodexMethodModelList    = "model/list"

	// 服务端 → 客户端（通知）
	CodexMethodInitialized = "initialized"
)

// CodexRPCRequest JSON-RPC 请求帧。
type CodexRPCRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// CodexRPCNotification JSON-RPC 通知帧（无 id）。
type CodexRPCNotification struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// CodexRPCResponse JSON-RPC 响应帧。
type CodexRPCResponse struct {
	ID     int              `json:"id"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *CodexRPCError   `json:"error,omitempty"`
}

// CodexRPCError JSON-RPC 错误。
type CodexRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CodexInitializeParams initialize 请求参数。
type CodexInitializeParams struct {
	ClientInfo   map[string]string   `json:"clientInfo"`
	Capabilities map[string]bool     `json:"capabilities"`
}

// CodexThreadStartParams thread/start 请求参数。
type CodexThreadStartParams struct {
	Model          string `json:"model,omitempty"`
	ApprovalPolicy string `json:"approvalPolicy,omitempty"`
	Sandbox        string `json:"sandbox,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
}

// CodexTurnStartParams turn/start 请求参数。
type CodexTurnStartParams struct {
	ThreadID string              `json:"threadId,omitempty"`
	Items    []CodexTurnInputItem `json:"items"`
}

// CodexTurnInputItem turn 输入项。
type CodexTurnInputItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
	Path string `json:"path,omitempty"`
}

// CodexEventTypes Codex 事件类型常量。
const (
	CodexEvtInitialized      = "initialized"
	CodexEvtThreadCreated    = "thread/created"
	CodexEvtTurnStarted      = "turn/started"
	CodexEvtTurnDelta        = "turn/delta"
	CodexEvtTurnCompleted    = "turn/completed"
	CodexEvtTurnFailed       = "turn/failed"
	CodexEvtItemAgentMessage = "item/agentMessage"
	CodexEvtItemReasoning    = "item/reasoning"
	CexEvtItemCommandCall    = "item/commandCall"
	CodexEvtItemCommandOutput = "item/commandOutput"
	CodexEvtItemFileChange   = "item/fileChange"
	CodexEvtItemQuestion     = "item/question"
)
