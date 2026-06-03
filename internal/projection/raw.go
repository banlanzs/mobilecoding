package projection

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/banlanzs/mobilecoding/internal/engine"
)

// Project 把引擎事件翻译为投影事件。深度解析 Claude stream-json。
func Project(in []engine.Event, sid string) []Event {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	out := make([]Event, 0, len(in))
	for _, ev := range in {
		switch ev.Kind {
		case engine.EventRaw:
			// 尝试深度解析 Claude stream-json
			parsed, err := parseClaudeEvent(ev.Data, sid)
			if err == nil {
				out = append(out, parsed)
			}
			// 解析失败的事件：不显示（避免原始 JSON 泄露到前端）
		case engine.EventLifecycle:
			// 只转发用户可见的生命周期事件
			if isUserVisibleLifecycle(ev.Message) {
				out = append(out, LifecycleEvent(sid, ev.Message))
			}
		}
	}
	return out
}

// isUserVisibleLifecycle 判断生命周期事件是否应该显示给用户。
func isUserVisibleLifecycle(msg string) bool {
	// 隐藏内部状态信息
	if strings.HasPrefix(msg, "cmd:") || strings.HasPrefix(msg, "ready:") ||
		strings.HasPrefix(msg, "started:") || strings.HasPrefix(msg, "exited") {
		return false
	}
	return true
}

// Stream 实时投影：从 input 读 engine.Event，输出 projection.Event。
// sessionID 由调用方传入。
func Stream(input <-chan engine.Event, output chan<- Event, sid string) {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	for ev := range input {
		switch ev.Kind {
		case engine.EventRaw:
			// 尝试深度解析 Claude stream-json
			parsed, err := parseClaudeEvent(ev.Data, sid)
			if err == nil {
				output <- parsed
			} else {
				output <- TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n"))
			}
		case engine.EventLifecycle:
			output <- LifecycleEvent(sid, ev.Message)
		}
	}
	close(output)
}

// parseClaudeEvent 深度解析 Claude stream-json 事件。
func parseClaudeEvent(data []byte, sid string) (Event, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Event{}, err
	}

	typ, _ := m["type"].(string)
	switch typ {
	case "assistant", "assistant_message":
		text, thinking := extractClaudeContent(m["message"])
		_ = thinking
		// 即使只有思考也发送事件（空文本也不跳过，保持 lastActivity 更新防超时）
		return TextEvent(sid, text), nil
	case "tool_use":
		name, _ := m["name"].(string)
		input := m["input"]
		return ToolUseEvent(sid, name, input), nil
	case "tool_result":
		name, _ := m["name"].(string)
		content := m["content"]
		return ToolResultEvent(sid, name, content), nil
	case "permission_request":
		toolName, _ := m["tool_name"].(string)
		prompt, _ := m["prompt"].(string)
		return PermissionRequestEvent(sid, toolName, prompt), nil
	case "plan_mode":
		return PlanModeEvent(sid, m), nil
	case "context_window":
		return ContextWindowEvent(sid, m), nil
	case "session", "system":
		// Claude 启动初始化事件，跳过
		return Event{}, errors.New("skip system/session event")
	case "result":
		// Claude 结束事件，跳过
		return Event{}, errors.New("skip result event")
	default:
		return Event{}, errors.New("unknown event type: " + typ)
	}
}

// extractClaudeContent 从 assistant_message 提取实际回复文本和思考内容。
// 返回 (responseText, thinkingText)
func extractClaudeContent(message any) (string, string) {
	switch v := message.(type) {
	case string:
		return v, ""
	case map[string]any:
		content, ok := v["content"]
		if !ok {
			return "", ""
		}
		contentArr, ok := content.([]any)
		if !ok {
			return "", ""
		}
		var responseParts []string
		var thinkingParts []string
		for _, block := range contentArr {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			text, _ := blockMap["text"].(string)
			thinking, _ := blockMap["thinking"].(string)

			if blockType == "text" && text != "" {
				responseParts = append(responseParts, text)
			}
			if blockType == "thinking" && thinking != "" {
				thinkingParts = append(thinkingParts, thinking)
			}
		}
		return strings.Join(responseParts, ""), strings.Join(thinkingParts, "")
	}
	return "", ""
}
