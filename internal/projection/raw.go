package projection

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/banlanzs/mobilecoding/internal/engine"
)

// PhaseTracker 追踪当前 Agent 阶段，用于发出配对的 start/end 事件。
// 必须跨 Project 调用共享，否则 tool_start/tool_end 等配对事件无法正确生成。
type PhaseTracker struct {
	inThinking   bool
	lastToolID   string
	lastToolName string // 记录最近工具名（用于 tool_result 配对）
}

// Project 把引擎事件翻译为投影事件。深度解析 Claude stream-json。
// pt 可选：传入则跨调用共享状态（推荐），不传则创建临时状态。
func Project(in []engine.Event, sid string, pt ...*PhaseTracker) []Event {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	var tracker *PhaseTracker
	if len(pt) > 0 && pt[0] != nil {
		tracker = pt[0]
	} else {
		tracker = &PhaseTracker{}
	}
	out := make([]Event, 0, len(in)*2)
	for _, ev := range in {
		switch ev.Kind {
		case engine.EventRaw:
			parsed, err := parseClaudeEvent(ev.Data, sid)
			if err == nil {
				out = tracker.emitLifecycle(out, parsed, sid)
			} else if !isJSON(ev.Data) {
				out = append(out, TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n")))
			}
		case engine.EventLifecycle:
			if isUserVisibleLifecycle(ev.Message) {
				out = append(out, LifecycleEvent(sid, ev.Message))
			}
		}
	}
	// 结束时关闭所有未配对的状态
	if tracker.inThinking {
		out = append(out, ThinkingEndEvent(sid))
		tracker.inThinking = false
	}
	return out
}

// emitLifecycle 根据事件类型追踪阶段转换并发出配对事件。
func (pt *PhaseTracker) emitLifecycle(out []Event, ev Event, sid string) []Event {
	switch ev.Type {
	case EventTextDelta:
		// thinking_delta 触发 thinking_start
		if ev.Thinking != "" && !pt.inThinking {
			out = append(out, ThinkingStartEvent(sid))
			pt.inThinking = true
		}
		// text_delta (无 thinking) 触发 thinking_end
		if ev.Thinking == "" && ev.Text != "" && pt.inThinking {
			out = append(out, ThinkingEndEvent(sid))
			pt.inThinking = false
		}
	case EventText:
		if pt.inThinking {
			out = append(out, ThinkingEndEvent(sid))
			pt.inThinking = false
		}
	case EventToolUse:
		if pt.inThinking {
			out = append(out, ThinkingEndEvent(sid))
			pt.inThinking = false
		}
		toolID := uuid.NewString()
		ev.ToolID = toolID
		pt.lastToolID = toolID
		pt.lastToolName = ev.ToolName
		if ev.ToolName == "Bash" {
			out = append(out, BashStartEvent(sid, toolID, formatToolInput(ev.ToolInput)))
		}
		out = append(out, ToolStartEvent(sid, toolID, ev.ToolName, ev.ToolInput))
	case EventToolResult:
		// 配对 end 事件：
		// 1) 优先用 pt.lastToolID（理想情况）
		// 2) 退化用 ev.ToolName（处理跨 Project 调用的情况）
		// 3) 最差也至少发一个 ToolEndEvent 避免前端状态卡住
		toolID := pt.lastToolID
		if toolID == "" {
			toolID = uuid.NewString()
		}
		toolName := ev.ToolName
		if toolName == "" {
			toolName = pt.lastToolName
		}
		if toolName == "Bash" {
			out = append(out, BashEndEvent(sid, toolID))
		}
		out = append(out, ToolEndEvent(sid, toolID, toolName))
		pt.lastToolID = ""
		pt.lastToolName = ""
	case EventTurnEnd:
		// Claude 整轮结束：清理所有悬挂状态
		if pt.inThinking {
			out = append(out, ThinkingEndEvent(sid))
			pt.inThinking = false
		}
		if pt.lastToolID != "" {
			toolName := pt.lastToolName
			if toolName == "Bash" {
				out = append(out, BashEndEvent(sid, pt.lastToolID))
			}
			out = append(out, ToolEndEvent(sid, pt.lastToolID, toolName))
			pt.lastToolID = ""
			pt.lastToolName = ""
		}
	}
	out = append(out, ev)
	return out
}

func formatToolInput(input any) string {
	if s, ok := input.(string); ok {
		return s
	}
	b, _ := json.Marshal(input)
	return string(b)
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
	// turn_end 是控制信号（来自 ClaudeRunner.waitLoop），
	// 用于在 Claude 异常退出导致 result 事件未发出时也能关闭前端的 turn 状态
	if strings.HasPrefix(msg, "turn_end") {
		return true
	}
	return true
}

// Stream 实时投影：从 input 读 engine.Event，输出 projection.Event。
// sessionID 由调用方传入。
// 使用共享的 phaseTracker 来正确生成 thinking/tool/bash 等配对事件。
func Stream(input <-chan engine.Event, output chan<- Event, sid string) {
	if sid == "" {
		sid = "sess_" + uuid.NewString()
	}
	tracker := &PhaseTracker{}
	for ev := range input {
		switch ev.Kind {
		case engine.EventRaw:
			parsed, err := parseClaudeEvent(ev.Data, sid)
			if err == nil {
				out := tracker.emitLifecycle(nil, parsed, sid)
				for _, pe := range out {
					output <- pe
				}
			} else if !isJSON(ev.Data) {
				output <- TextEvent(sid, strings.TrimRight(string(ev.Data), "\r\n"))
			}
		case engine.EventLifecycle:
			if isUserVisibleLifecycle(ev.Message) {
				output <- LifecycleEvent(sid, ev.Message)
			}
		}
	}
	// 关闭时清理悬挂状态
	if tracker.inThinking {
		output <- ThinkingEndEvent(sid)
		tracker.inThinking = false
	}
	if tracker.lastToolID != "" {
		toolName := tracker.lastToolName
		if toolName == "Bash" {
			output <- BashEndEvent(sid, tracker.lastToolID)
		}
		output <- ToolEndEvent(sid, tracker.lastToolID, toolName)
		tracker.lastToolID = ""
		tracker.lastToolName = ""
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
	case "control_request":
		// Claude stdio permission tool 协议：
		//   {"type":"control_request","request_id":"...","request":{"tool_name":"Bash","input":{...}}}
		// 前端按 tool_name/prompt 显示 Allow/Deny，再通过 control_response 回传。
		reqID, _ := m["request_id"].(string)
		if reqID == "" {
			reqID = newMessageID()
		}
		req, _ := m["request"].(map[string]any)
		toolName, prompt := extractControlRequestInfo(req)
		return PermissionAskEvent(sid, reqID, toolName, prompt), nil
	case "plan_mode":
		return PlanModeEvent(sid, m), nil
	case "context_window":
		return ContextWindowEvent(sid, m), nil
	case "session", "system":
		// Claude 启动初始化事件，跳过
		return Event{}, errors.New("skip system/session event")
	case "result":
		// Claude 整轮结束信号：发 turn_end 让前端把按钮从"中止"切回"发送"
		resultStr, _ := m["result"].(string)
		if resultStr == "" {
			if msg, ok := m["message"].(string); ok {
				resultStr = msg
			}
		}
		subtype, _ := m["subtype"].(string)
		if subtype == "" {
			subtype = "success"
		}
		isError := false
		if isErr, ok := m["is_error"].(bool); ok {
			isError = isErr
		}
		ev := TurnEndEvent(sid, resultStr, isError)
		ev.Message = subtype + ": " + resultStr
		return ev, nil
	default:
		return Event{}, errors.New("unknown event type: " + typ)
	}
}

// extractControlRequestInfo 从 Claude control_request.request 中提取工具名和提示。
func extractControlRequestInfo(req map[string]any) (string, string) {
	if req == nil {
		return "", ""
	}
	toolName, _ := req["tool_name"].(string)
	if toolName == "" {
		// 部分版本使用 nested 字段
		if sub, ok := req["toolName"].(string); ok {
			toolName = sub
		}
	}
	prompt, _ := req["prompt"].(string)
	if prompt == "" {
		// fallback：使用 input.command 或 input.description 作为提示
		if input, ok := req["input"].(map[string]any); ok {
			switch v := input["command"].(type) {
			case string:
				prompt = "执行: " + truncateForPrompt(v, 200)
			default:
				if d, ok := input["description"].(string); ok {
					prompt = d
				}
			}
		}
	}
	return toolName, prompt
}

// truncateForPrompt 截断 prompt 内容。
func truncateForPrompt(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
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
