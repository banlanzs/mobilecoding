// Package ws 实现 mytool WebSocket 协议（v1，JSON 编码）。
// 协议细节见 spec §4。
package ws

import "encoding/json"

type Envelope struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	OK        *bool           `json:"ok,omitempty"`
	Error     *RPCError       `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Event     json.RawMessage `json:"event,omitempty"`
}

type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
