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

	// 累积中的 tool_use 块（来自 content_block_start / content_block_delta 序列）。
	// key 是 content_block 的 index，值是该 block 的元数据 + 累积的 input JSON。
	// 在 content_block_stop 时统一 emit ToolUseEvent。
	pendingToolUses map[int]*pendingToolUse

	// tool_use_id -> tool_name，用于把后续 user 消息里的 tool_result 配对回正确工具名。
	toolUseIDs map[string]string
}

// pendingToolUse 跟踪一个尚未完成的 content_block（type=tool_use）。
// 真正的 ToolUseEvent 在 content_block_stop 时发出。
type pendingToolUse struct {
	ID         string // 来自 content_block.id
	Name       string // 工具名，如 "Bash"
	InputJSON  string // 累积的 input_json_delta
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
		tracker = &PhaseTracker{
			pendingToolUses: make(map[int]*pendingToolUse),
			toolUseIDs:      make(map[string]string),
		}
	}
	// 兼容历史调用方传入的半初始 PhaseTracker
	if tracker.pendingToolUses == nil {
		tracker.pendingToolUses = make(map[int]*pendingToolUse)
	}
	if tracker.toolUseIDs == nil {
		tracker.toolUseIDs = make(map[string]string)
	}
	out := make([]Event, 0, len(in)*2)
	for _, ev := range in {
		switch ev.Kind {
		case engine.EventRaw:
			events, err := parseClaudeEventWithTracker(ev.Data, sid, tracker)
			if err == nil {
				out = tracker.emitLifecycleBatch(out, events, sid)
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

// emitLifecycleBatch 处理多事件批次（来自 parseClaudeEventWithTracker）。
// 对每个事件做阶段追踪并追加（pair 事件 + 事件本身）。
func (pt *PhaseTracker) emitLifecycleBatch(out []Event, events []Event, sid string) []Event {
	for _, ev := range events {
		out = pt.emitLifecycle(out, ev, sid)
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
		// 记录 tool_use_id → name 映射（用于把后续 user 消息的 tool_result 配对回正确工具名）
		if ev.ToolID != "" {
			pt.toolUseIDs[ev.ToolID] = ev.ToolName
		}
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
	tracker := &PhaseTracker{
		pendingToolUses: make(map[int]*pendingToolUse),
		toolUseIDs:      make(map[string]string),
	}
	for ev := range input {
		switch ev.Kind {
		case engine.EventRaw:
			events, err := parseClaudeEventWithTracker(ev.Data, sid, tracker)
			if err == nil {
				out := tracker.emitLifecycleBatch(nil, events, sid)
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

// parseClaudeEvent 深度解析 Claude stream-json 事件（无状态版本，保留向后兼容）。
// 建议使用 parseClaudeEventWithTracker 以支持 tool_use 跨事件累积。
func parseClaudeEvent(data []byte, sid string) (Event, error) {
	events, err := parseClaudeEventWithTracker(data, sid, nil)
	if err != nil {
		return Event{}, err
	}
	if len(events) == 0 {
		return Event{}, errors.New("parse produced no events")
	}
	return events[0], nil
}

// parseClaudeEventWithTracker 深度解析 Claude stream-json 事件。
// 返回多个事件（部分 stream-json 事件需要累积 input 或聚合多个 content block）。
// 真实 Claude stream-json 格式：
//   {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_01","name":"Bash","input":{}}}
//   {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\":"}}
//   {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"ls\"}"}}
//   {"type":"content_block_stop","index":1}
//   {"type":"message_stop"}
//   {"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_01","content":"..."}]}}
// pt 用于跨事件累积 tool_use 的 input_json_delta；传 nil 时不支持 content_block 多步累积。
func parseClaudeEventWithTracker(data []byte, sid string, pt *PhaseTracker) ([]Event, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	typ, _ := m["type"].(string)
	switch typ {
	case "assistant", "assistant_message":
			text, thinking := extractClaudeContent(m["message"])
			if text == "" && thinking == "" {
				return nil, errors.New("empty message")
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
				// 仅思考阶段：发送 text_delta 携带 thinking 内容，前端可折叠显示
				if thinking != "" {
					ev := TextDeltaEvent(sid, "", 0)
					ev.Thinking = thinking
					return []Event{ev}, nil
				}
				return nil, errors.New("empty message")
			}
			if thinking != "" {
				return []Event{TextEventWithThinking(sid, text, thinking)}, nil
			}
			return []Event{TextEvent(sid, text)}, nil
	case "content_block_start":
		return handleContentBlockStart(m, sid, pt)
	case "content_block_delta":
		return handleContentBlockDelta(m, sid, pt)
	case "content_block_stop":
		return handleContentBlockStop(m, sid, pt)
	case "message_start", "message_delta", "message_stop", "ping":
		// 控制事件：跳过
		return nil, errors.New("skip " + typ)
	case "user":
		// 工具结果通常在 user 消息中：{"message":{"role":"user","content":[{"type":"tool_result",...}]}}
		return handleUserMessage(m, sid, pt)
	case "tool_use":
		// 兼容老格式（非标准 stream-json，但 claude_code 早期版本会这样发）
		name, _ := m["name"].(string)
		input := m["input"]
		return []Event{ToolUseEvent(sid, name, input)}, nil
	case "tool_result":
		// 兼容老格式
		name, _ := m["name"].(string)
		content := m["content"]
		return []Event{ToolResultEvent(sid, name, content)}, nil
	case "permission_request":
		toolName, _ := m["tool_name"].(string)
		prompt, _ := m["prompt"].(string)
		return []Event{PermissionRequestEvent(sid, toolName, prompt)}, nil
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
		return []Event{PermissionAskEvent(sid, reqID, toolName, prompt)}, nil
	case "plan_mode":
		return []Event{PlanModeEvent(sid, m)}, nil
	case "context_window":
		return []Event{ContextWindowEvent(sid, m)}, nil
	case "session", "system":
		// Claude 启动初始化事件，跳过
		return nil, errors.New("skip system/session event")
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
		return []Event{ev}, nil
	default:
		return nil, errors.New("unknown event type: " + typ)
	}
}

// handleContentBlockStart 处理 content_block_start 事件。
// 对 tool_use 类型：登记 pendingToolUses，暂不 emit（等 content_block_stop 拿到完整 input 后再 emit）。
func handleContentBlockStart(m map[string]any, sid string, pt *PhaseTracker) ([]Event, error) {
	cb, ok := m["content_block"].(map[string]any)
	if !ok {
		return nil, errors.New("skip content_block_start: no content_block")
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
			// 真实 Claude stream-json：text 块的 content_block_start 通常 text 为空，
			// 实际内容由后续 content_block_delta 填充。直接跳过这个 start 事件即可。
			return nil, nil
		}
		return []Event{TextDeltaEvent(sid, text, blockIndex)}, nil
	case "thinking":
		think, _ := cb["thinking"].(string)
		if think == "" {
			return nil, errors.New("empty content_block_start thinking")
		}
		ev := TextDeltaEvent(sid, "", blockIndex)
		ev.Thinking = think
		return []Event{ev}, nil
	case "tool_use":
		// 登记：等 stop 时再统一 emit
		if pt != nil {
			id, _ := cb["id"].(string)
			name, _ := cb["name"].(string)
			pt.pendingToolUses[blockIndex] = &pendingToolUse{
				ID:   id,
				Name: name,
			}
		}
		return nil, nil
	default:
		return nil, errors.New("skip content_block_start: " + cbType)
	}
}

// handleContentBlockDelta 处理 content_block_delta 事件。
// 对 input_json_delta：累积到 pendingToolUses[index].InputJSON（不会立即 emit）。
// 对 text_delta / thinking_delta：直接 emit TextDeltaEvent。
func handleContentBlockDelta(m map[string]any, sid string, pt *PhaseTracker) ([]Event, error) {
	delta, ok := m["delta"].(map[string]any)
	if !ok {
		return nil, errors.New("missing delta")
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
			return nil, errors.New("empty text delta")
		}
		// 剥离内联 <think> 标签
		think, cleanText := stripThinkTags(text)
		if cleanText == "" && think != "" {
			return []Event{TextDeltaEvent(sid, think, blockIndex)}, nil
		}
		if cleanText == "" {
			return nil, errors.New("empty after stripping think tags")
		}
		ev := TextDeltaEvent(sid, cleanText, blockIndex)
		if think != "" {
			ev.Thinking = think
		}
		return []Event{ev}, nil

	case "thinking_delta":
		thinkText, _ := delta["thinking"].(string)
		if thinkText == "" {
			return nil, errors.New("empty thinking delta")
		}
		// 将 thinking delta 作为 text_delta 发送（前端会渲染在思考区域）
		ev := TextDeltaEvent(sid, "", blockIndex)
		ev.Thinking = thinkText
		return []Event{ev}, nil

	case "input_json_delta":
		// 累积 tool_use 的 partial JSON
		if pt != nil {
			partial, _ := delta["partial_json"].(string)
			if pending, ok := pt.pendingToolUses[blockIndex]; ok && partial != "" {
				pending.InputJSON += partial
			}
		}
		return nil, nil

	default:
		return nil, errors.New("skip non-text delta: " + deltaType)
	}
}

// handleContentBlockStop 处理 content_block_stop 事件。
// 对 tool_use：解析累积的 input JSON，emit ToolUseEvent（让 emitLifecycle 进一步配对 start 事件）。
// 同时把 pendingToolUses[index] 从 map 中移除。
func handleContentBlockStop(m map[string]any, sid string, pt *PhaseTracker) ([]Event, error) {
	blockIndex := 0
	if idx, ok := m["index"].(float64); ok {
		blockIndex = int(idx)
	}
	if pt == nil {
		return nil, nil
	}
	pending, ok := pt.pendingToolUses[blockIndex]
	if !ok {
		// text/thinking 块的 stop 不需要 emit
		return nil, nil
	}
	delete(pt.pendingToolUses, blockIndex)

	input := parseToolInputJSON(pending.InputJSON)
	// 把 tool_use_id 写入 ToolUseEvent.ToolID 字段，供 emitLifecycle 建立 toolUseIDs 映射
	// 同时在 stop 时直接建立映射（避免依赖 emitLifecycle 顺序）
	if pending.ID != "" && pending.Name != "" {
		pt.toolUseIDs[pending.ID] = pending.Name
	}
	ev := ToolUseEvent(sid, pending.Name, input)
	ev.ToolID = pending.ID
	return []Event{ev}, nil
}

// handleUserMessage 处理 user 消息：从中提取 tool_result content blocks 并 emit ToolResultEvent。
// 真实 Claude 格式：{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"...","content":"..."}]}}
func handleUserMessage(m map[string]any, sid string, pt *PhaseTracker) ([]Event, error) {
	msg, _ := m["message"].(map[string]any)
	if msg == nil {
		return nil, errors.New("user message without message field")
	}
	content, ok := msg["content"].([]any)
	if !ok {
		return nil, errors.New("user message without content array")
	}
	var out []Event
	for _, block := range content {
		bm, ok := block.(map[string]any)
		if !ok {
			continue
		}
		blockType, _ := bm["type"].(string)
		if blockType != "tool_result" {
			continue
		}
		toolUseID, _ := bm["tool_use_id"].(string)
		toolResultContent := bm["content"]
		// 用 toolUseID 查回工具名（建立更准确的 ToolResult 配对）
		toolName := ""
		if pt != nil && toolUseID != "" {
			toolName = pt.toolUseIDs[toolUseID]
		}
		out = append(out, ToolResultEvent(sid, toolName, toolResultContent))
	}
	if len(out) == 0 {
		return nil, errors.New("user message without tool_result blocks")
	}
	return out, nil
}

// parseToolInputJSON 解析累积的 partial JSON 为 any。空时返回空 map。
func parseToolInputJSON(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		// 解析失败时返回原始字符串，避免丢信息
		return s
	}
	return v
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
