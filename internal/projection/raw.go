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
			} else {
				// 非 Claude JSON，透传为文本
				out = append(out, TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n")))
			}
		case engine.EventLifecycle:
			out = append(out, LifecycleEvent(sid, ev.Message))
		}
	}
	return out
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
	case "assistant_message":
		msg := extractClaudeAssistantText(m["message"])
		return TextEvent(sid, msg), nil
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
	case "session":
		return SessionEvent(sid, m), nil
	case "system":
		// Claude 启动初始化事件，跳过
		return Event{}, errors.New("skip system event")
	case "result":
		// Claude 结束事件，跳过
		return Event{}, errors.New("skip result event")
	default:
		return Event{}, errors.New("unknown event type: " + typ)
	}
}

// extractClaudeAssistantText 从 Claude assistant_message 的 message 字段提取文本。
// message 可能是字符串（旧格式）或对象（新格式，含 content 数组）。
func extractClaudeAssistantText(message any) string {
	switch v := message.(type) {
	case string:
		return v
	case map[string]any:
		content, ok := v["content"]
		if !ok {
			return ""
		}
		contentArr, ok := content.([]any)
		if !ok {
			return ""
		}
		var parts []string
		for _, block := range contentArr {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			text, _ := blockMap["text"].(string)
			if blockType == "text" && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}
