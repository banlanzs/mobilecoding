package projection

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

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
			} else if !isJSON(ev.Data) {
				// 非 JSON 数据：透传为文本
				out = append(out, TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n")))
			}
			// JSON 但无法解析的：跳过（避免原始 JSON 泄露到前端）
		case engine.EventLifecycle:
			// 只转发用户可见的生命周期事件
			if isUserVisibleLifecycle(ev.Message) {
				out = append(out, LifecycleEvent(sid, ev.Message))
			}
		}
	}
	return out
}

// isJSON 判断数据是否为有效 JSON。
func isJSON(data []byte) bool {
	var m map[string]any
	return json.Unmarshal(data, &m) == nil
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
			parsed, err := parseClaudeEvent(ev.Data, sid)
			if err == nil {
				output <- parsed
			} else if !isJSON(ev.Data) {
				output <- TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n"))
			}
		case engine.EventLifecycle:
			if isUserVisibleLifecycle(ev.Message) {
				output <- LifecycleEvent(sid, ev.Message)
			}
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
			if text == "" && thinking == "" {
				return Event{}, errors.New("empty message")
			}
			// 剥离内联 <think> 标签
			if strippedThink, strippedText := stripThinkTags(text); strippedThink != "" {
				if thinking != "" {
					thinking = thinking + "\n\n" + strippedThink
				} else {
					thinking = strippedThink
				}
				text = strippedText
			}
			if text == "" {
				// 仅思考阶段：不暴露内容，只显示简短指示器
				if thinking != "" {
					return LifecycleEvent(sid, "思考中…"), nil
				}
				return Event{}, errors.New("empty message")
			}
			if thinking != "" {
				return TextEventWithThinking(sid, text, thinking), nil
			}
			return TextEvent(sid, text), nil
	case "content_block_start":
		// 记录文本块开始，不做展示
		cb, ok := m["content_block"].(map[string]any)
		if !ok {
			return Event{}, errors.New("skip content_block_start: no content_block")
		}
		cbType, _ := cb["type"].(string)
		blockIndex := 0
		if idx, ok := m["index"].(float64); ok {
			blockIndex = int(idx)
		}
		switch cbType {
		case "text":
			text, _ := cb["text"].(string)
			if text == "" {
				return Event{}, errors.New("empty content_block_start text")
			}
			return TextDeltaEvent(sid, text, blockIndex), nil
		case "thinking":
			think, _ := cb["thinking"].(string)
			if think == "" {
				return Event{}, errors.New("empty content_block_start thinking")
			}
			ev := TextDeltaEvent(sid, "", blockIndex)
			ev.Thinking = think
			return ev, nil
		default:
			return Event{}, errors.New("skip content_block_start: " + cbType)
		}
	case "content_block_delta":
		delta, ok := m["delta"].(map[string]any)
		if !ok {
			return Event{}, errors.New("missing delta")
		}
		deltaType, _ := delta["type"].(string)
		blockIndex := 0
		if idx, ok := m["index"].(float64); ok {
			blockIndex = int(idx)
		}

		switch deltaType {
		case "text_delta":
			text, _ := delta["text"].(string)
			if text == "" {
				return Event{}, errors.New("empty text delta")
			}
			// 剥离内联 <think> 标签
			think, cleanText := stripThinkTags(text)
			if cleanText == "" && think != "" {
				return TextDeltaEvent(sid, think, blockIndex), nil
			}
			if cleanText == "" {
				return Event{}, errors.New("empty after stripping think tags")
			}
			ev := TextDeltaEvent(sid, cleanText, blockIndex)
			if think != "" {
				ev.Thinking = think
			}
			return ev, nil

		case "thinking_delta":
			thinkText, _ := delta["thinking"].(string)
			if thinkText == "" {
				return Event{}, errors.New("empty thinking delta")
			}
			// 将 thinking delta 作为 text_delta 发送（前端会渲染在思考区域）
			ev := TextDeltaEvent(sid, "", blockIndex)
			ev.Thinking = thinkText
			return ev, nil

		default:
			return Event{}, errors.New("skip non-text delta")
		}
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

// truncateThinking 截断思考内容用于生命周期事件显示。
func truncateThinking(thinking string) string {
	if thinking == "" {
		return "..."
	}
	const maxLen = 80
	if utf8.RuneCountInString(thinking) > maxLen {
		runes := []rune(thinking)
		return string(runes[:maxLen]) + "..."
	}
	return thinking
}

// stripThinkTags 从文本中剥离 <think>...</think> 标签。
// 返回 (thinking内容, 清理后的文本)。
func stripThinkTags(text string) (string, string) {
	const open = "<think>"
	const close = "</think>"

	var thinkingParts []string
	result := text

	for {
		start := strings.Index(result, open)
		if start == -1 {
			break
		}
		end := strings.Index(result, close)
		if end == -1 || end < start+len(open) {
			break
		}
		thinkingParts = append(thinkingParts, result[start+len(open):end])
		result = result[:start] + result[end+len(close):]
	}

	return strings.TrimSpace(strings.Join(thinkingParts, "\n\n")), strings.TrimSpace(result)
}
