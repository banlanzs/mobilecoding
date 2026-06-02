package projection

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/jaycrl/mytool/internal/engine"
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
		msg, _ := m["message"].(string)
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
	default:
		return Event{}, errors.New("unknown event type")
	}
}
