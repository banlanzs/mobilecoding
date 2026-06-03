// Package relay 实现 WebSocket 中继服务器，支持 agent（CLI）和 client（手机）双向消息转发。
package relay

import "time"

const (
	Version = 1

	// Agent 帧类型
	TypeAgentRegister   = "agent.register"
	TypeAgentRegistered = "agent.registered"
	TypeAgentReconnect  = "agent.reconnect"

	// Client 帧类型
	TypeClientPair      = "client.pair"
	TypeClientReconnect = "client.reconnect"
	TypeClientPaired    = "client.paired"
	TypeClientAttached  = "client.attached"

	// 中继帧类型
	TypeRelayForward = "relay.forward"
	TypeRelayError   = "relay.error"
	TypeRelayPing    = "relay.ping"
	TypeRelayPong    = "relay.pong"

	// 方向
	DirectionClientToAgent = "client_to_agent"
	DirectionAgentToClient = "agent_to_client"

	// 内容类型
	ContentTypeMobileCoding = "mobilecoding.ws.v1"
)

// 错误码
const (
	CodePairingRejected   = "pairing_rejected"
	CodeUnauthorized      = "unauthorized"
	CodeCapacityReached   = "capacity_reached"
	CodeTimeout           = "timeout"
	CodeFrameTooLarge     = "frame_too_large"
	CodeProtocolError     = "protocol_error"
	CodeTargetUnavailable = "target_unavailable"
	CodeQueueFull         = "queue_full"
	CodeAgentDisconnected = "agent_disconnected"
)

// ControlFrame 是所有控制帧的基础结构。
type ControlFrame struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
}

// AgentRegisterFrame 是 agent 注册请求。
type AgentRegisterFrame struct {
	Type              string `json:"type"`
	Version           int    `json:"version"`
	SessionID         string `json:"sessionId"`
	PairingSecretHash string `json:"pairingSecretHash"`
}

// AgentRegisteredFrame 是 agent 注册成功响应。
type AgentRegisteredFrame struct {
	Type      string `json:"type"`
	Version   int    `json:"version"`
	SessionID string `json:"sessionId"`
}

// ClientPairFrame 是 client 配对请求。
type ClientPairFrame struct {
	Type          string `json:"type"`
	Version       int    `json:"version"`
	SessionID     string `json:"sessionId"`
	PairingSecret string `json:"pairingSecret"`
	DeviceName    string `json:"deviceName,omitempty"`
}

// ClientPairedFrame 是 client 配对成功响应。
type ClientPairedFrame struct {
	Type      string `json:"type"`
	Version   int    `json:"version"`
	SessionID string `json:"sessionId"`
	ClientID  string `json:"clientId"`
}

// ClientAttachedFrame 通知 agent 有 client 已连接。
type ClientAttachedFrame struct {
	Type      string `json:"type"`
	Version   int    `json:"version"`
	SessionID string `json:"sessionId"`
	ClientID  string `json:"clientId"`
}

// ForwardEnvelope 是转发消息的信封。
type ForwardEnvelope struct {
	Type        string `json:"type"`
	Version     int    `json:"version"`
	SessionID   string `json:"sessionId"`
	ClientID    string `json:"clientId,omitempty"`
	Direction   string `json:"direction"`
	MessageID   string `json:"messageId"`
	ContentType string `json:"contentType"`
	Payload     string `json:"payload"` // JSON 编码的消息内容
}

// ErrorFrame 是错误帧。
type ErrorFrame struct {
	Type    string `json:"type"`
	Version int    `json:"version"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PairingReadyEvent 是配对就绪事件，写入文件供 CLI 读取。
type PairingReadyEvent struct {
	Type      string `json:"type"`
	RelayURL  string `json:"relayUrl"`
	SessionID string `json:"sessionId"`
	Secret    string `json:"secret"`
	ExpiresAt int64  `json:"expiresAt"`
}

// SessionConfig 是会话配置。
type SessionConfig struct {
	// AgentGracePeriod 是 agent 断开后的重连宽限期。
	AgentGracePeriod time.Duration
	// PairingTTL 是配对码的有效期。
	PairingTTL time.Duration
	// MaxPayloadBytes 是最大 payload 大小。
	MaxPayloadBytes int
	// MaxConnections 是最大连接数。
	MaxConnections int
}

// DefaultSessionConfig 返回默认配置。
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		AgentGracePeriod: 30 * time.Second,
		PairingTTL:       5 * time.Minute,
		MaxPayloadBytes:  1024 * 1024, // 1MB
		MaxConnections:   100,
	}
}

// NewErrorFrame 创建错误帧。
func NewErrorFrame(code string) ErrorFrame {
	return ErrorFrame{
		Type:    TypeRelayError,
		Version: Version,
		Code:    code,
		Message: defaultErrorMessage(code),
	}
}

func defaultErrorMessage(code string) string {
	switch code {
	case CodePairingRejected:
		return "pairing rejected"
	case CodeUnauthorized:
		return "unauthorized"
	case CodeCapacityReached:
		return "capacity reached"
	case CodeTimeout:
		return "timeout"
	case CodeFrameTooLarge:
		return "frame too large"
	case CodeProtocolError:
		return "protocol error"
	case CodeTargetUnavailable:
		return "target unavailable"
	case CodeQueueFull:
		return "queue full"
	case CodeAgentDisconnected:
		return "agent disconnected"
	default:
		return "protocol error"
	}
}
