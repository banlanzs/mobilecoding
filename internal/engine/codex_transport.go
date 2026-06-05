package engine

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ParseCodexJSONRPC 解析 codex app-server 的 JSON-RPC 帧。
// 参考 easycodex session-orchestrator.ts 的消息规范化逻辑。
// 返回 Event{Kind: EventRaw, Data: <json bytes>}。
func ParseCodexJSONRPC(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{Kind: EventRaw, Data: nil}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return Event{}, errors.New("invalid JSON: " + err.Error())
	}
	cp := make([]byte, len(line))
	copy(cp, line)
	return Event{Kind: EventRaw, Data: cp}, nil
}

// NormalizeCodexEvent 将 Codex JSON-RPC 事件规范化为 projection 友好的格式。
// 参考 easycodex session-orchestrator.ts 的 summarizeMessageForMobile。
func NormalizeCodexEvent(data []byte) (eventType string, content map[string]any, err error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return "", nil, fmt.Errorf("parse codex event: %w", err)
	}

	method, _ := m["method"].(string)
	params, _ := m["params"].(map[string]any)

	switch method {
	case CodexEvtInitialized:
		return "initialized", params, nil
	case CodexEvtThreadCreated:
		return "thread_created", params, nil
	case CodexEvtTurnStarted:
		return "turn_started", params, nil
	case CodexEvtTurnDelta:
		return "turn_delta", params, nil
	case CodexEvtTurnCompleted:
		return "turn_completed", params, nil
	case CodexEvtTurnFailed:
		return "turn_failed", params, nil
	case CodexEvtItemAgentMessage:
		return "agent_message", params, nil
	case CodexEvtItemReasoning:
		return "reasoning", params, nil
	case CexEvtItemCommandCall:
		return "command_call", params, nil
	case CodexEvtItemCommandOutput:
		return "command_output", params, nil
	case CodexEvtItemFileChange:
		return "file_change", params, nil
	case CodexEvtItemQuestion:
		return "question", params, nil
	default:
		// 响应帧（有 id + result/error）
		if _, ok := m["id"]; ok {
			return "rpc_response", m, nil
		}
		return "unknown", m, nil
	}
}
