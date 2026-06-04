package projection

import (
	"testing"

	"github.com/banlanzs/mobilecoding/internal/engine"
)

func TestProjectUsesProvidedSessionID(t *testing.T) {
	in := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte("hello\n")},
	}
	got := Project(in, "sess_fixed_42")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].SessionID != "sess_fixed_42" {
		t.Errorf("SessionID = %q, want sess_fixed_42", got[0].SessionID)
	}
}

func TestProjectFallsBackToUUID(t *testing.T) {
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte("hi")}}
	got := Project(in, "")
	if got[0].SessionID == "" {
		t.Error("Project should generate uuid when sid is empty")
	}
	if got[0].SessionID[:5] != "sess_" {
		t.Errorf("generated sid should start with sess_, got %q", got[0].SessionID)
	}
}

func TestTextEvent(t *testing.T) {
	e := TextEvent("s1", "hello")
	if e.Type != EventText {
		t.Errorf("Type = %q, want text", e.Type)
	}
	if e.SessionID != "s1" || e.Text != "hello" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestLifecycleEvent(t *testing.T) {
	e := LifecycleEvent("s1", "started")
	if e.Type != EventLifecycle {
		t.Errorf("Type = %q, want lifecycle", e.Type)
	}
	if e.SessionID != "s1" || e.Message != "started" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestStreamWithSID(t *testing.T) {
	in := make(chan engine.Event, 2)
	in <- engine.Event{Kind: engine.EventRaw, Data: []byte("a\n")}
	in <- engine.Event{Kind: engine.EventLifecycle, Message: "x"}
	close(in)
	out := make(chan Event, 4)
	Stream(in, out, "sess_stream")
	var got []Event
	for e := range out {
		got = append(got, e)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].SessionID != "sess_stream" || got[1].SessionID != "sess_stream" {
		t.Errorf("SessionIDs should both be sess_stream, got %q / %q", got[0].SessionID, got[1].SessionID)
	}
}

func TestProjectClaudeToolUse(t *testing.T) {
	data := `{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	// emitLifecycle 现在为 Bash 工具额外发出 BashStartEvent + ToolStartEvent 配对事件
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (BashStart + ToolStart + ToolUse)", len(got))
	}
	if got[0].Type != EventBashStart {
		t.Errorf("got[0].Type = %q, want bash_start", got[0].Type)
	}
	if got[1].Type != EventToolStart {
		t.Errorf("got[1].Type = %q, want tool_start", got[1].Type)
	}
	if got[2].Type != EventToolUse {
		t.Errorf("got[2].Type = %q, want tool_use", got[2].Type)
	}
	if got[2].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", got[2].ToolName)
	}
}

func TestProjectClaudeToolResult(t *testing.T) {
	data := `{"type":"tool_result","name":"Bash","content":"ok"}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	// 没有 lastToolID 时仍然发出 BashEnd + ToolEnd + ToolResult
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Type != EventBashEnd {
		t.Errorf("got[0].Type = %q, want bash_end", got[0].Type)
	}
	if got[1].Type != EventToolEnd {
		t.Errorf("got[1].Type = %q, want tool_end", got[1].Type)
	}
	if got[2].Type != EventToolResult {
		t.Errorf("got[2].Type = %q, want tool_result", got[2].Type)
	}
}

func TestProjectClaudePermissionRequest(t *testing.T) {
	data := `{"type":"permission_request","tool_name":"Bash","prompt":"Allow?"}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != EventPermissionReq {
		t.Errorf("Type = %q, want permission_request", got[0].Type)
	}
	if got[0].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", got[0].ToolName)
	}
}

func TestProjectClaudeControlRequest(t *testing.T) {
	// Claude stdio permission tool 协议：control_request
	data := `{"type":"control_request","request_id":"req_123","request":{"tool_name":"Bash","input":{"command":"ls"},"prompt":"Allow Bash?"}}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != EventPermissionAsk {
		t.Errorf("Type = %q, want permission_ask", got[0].Type)
	}
	if got[0].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", got[0].ToolName)
	}
	if got[0].MessageID != "req_123" {
		t.Errorf("MessageID = %q, want req_123 (用作 requestId)", got[0].MessageID)
	}
}

func TestProjectClaudeResult(t *testing.T) {
	data := `{"type":"result","subtype":"success","is_error":false,"result":"Hello world","duration_ms":1234}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != EventTurnEnd {
		t.Errorf("Type = %q, want turn_end", got[0].Type)
	}
	if got[0].Text != "Hello world" {
		t.Errorf("Text = %q, want Hello world", got[0].Text)
	}
}

func TestProjectPhaseTrackerSharedState(t *testing.T) {
	// 验证跨 Project 调用共享 phaseTracker 能正确配对 start/end 事件
	tracker := &PhaseTracker{}
	useData := `{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`
	resData := `{"type":"tool_result","name":"Bash","content":"ok"}`

	got1 := Project([]engine.Event{{Kind: engine.EventRaw, Data: []byte(useData)}}, "sess_test", tracker)
	got2 := Project([]engine.Event{{Kind: engine.EventRaw, Data: []byte(resData)}}, "sess_test", tracker)

	// 第一次：tool_use → BashStart + ToolStart + ToolUse (3 个)
	if len(got1) != 3 {
		t.Fatalf("got1 len = %d, want 3", len(got1))
	}
	if got1[0].Type != EventBashStart {
		t.Errorf("got1[0].Type = %q, want bash_start", got1[0].Type)
	}
	if got1[1].Type != EventToolStart {
		t.Errorf("got1[1].Type = %q, want tool_start", got1[1].Type)
	}
	if got1[2].Type != EventToolUse {
		t.Errorf("got1[2].Type = %q, want tool_use", got1[2].Type)
	}

	// 第二次：tool_result → BashEnd + ToolEnd + ToolResult (3 个)
	if len(got2) != 3 {
		t.Fatalf("got2 len = %d, want 3", len(got2))
	}
	if got2[0].Type != EventBashEnd {
		t.Errorf("got2[0].Type = %q, want bash_end", got2[0].Type)
	}
	if got2[1].Type != EventToolEnd {
		t.Errorf("got2[1].Type = %q, want tool_end", got2[1].Type)
	}
	if got2[2].Type != EventToolResult {
		t.Errorf("got2[2].Type = %q, want tool_result", got2[2].Type)
	}

	// 验证 end 事件的 toolID 和 start 事件的 toolID 一致
	if got1[0].ToolID != got2[0].ToolID {
		t.Errorf("BashStart.ToolID = %q != BashEnd.ToolID = %q (应该一致)", got1[0].ToolID, got2[0].ToolID)
	}
	if got1[1].ToolID != got2[1].ToolID {
		t.Errorf("ToolStart.ToolID = %q != ToolEnd.ToolID = %q (应该一致)", got1[1].ToolID, got2[1].ToolID)
	}
}

func TestProjectTurnEndCleansHangingState(t *testing.T) {
	// 验证 result 事件触发悬挂状态清理
	tracker := &PhaseTracker{}
	useData := `{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`
	resData := `{"type":"result","subtype":"success","is_error":false,"result":"Done"}`

	// 1. tool_use（没有对应的 tool_result）
	_ = Project([]engine.Event{{Kind: engine.EventRaw, Data: []byte(useData)}}, "sess_test", tracker)

	// 2. result 直接到达，没有 tool_result
	got := Project([]engine.Event{{Kind: engine.EventRaw, Data: []byte(resData)}}, "sess_test", tracker)

	// 应该生成 BashEnd + ToolEnd 来清理悬挂状态
	foundBashEnd := false
	foundToolEnd := false
	for _, e := range got {
		if e.Type == EventBashEnd {
			foundBashEnd = true
		}
		if e.Type == EventToolEnd {
			foundToolEnd = true
		}
	}
	if !foundBashEnd {
		t.Error("result should trigger BashEnd to clean hanging state")
	}
	if !foundToolEnd {
		t.Error("result should trigger ToolEnd to clean hanging state")
	}
}

func TestProjectClaudeAssistantMessage(t *testing.T) {
	data := `{"type":"assistant_message","message":"Hello"}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Type != EventText {
		t.Errorf("Type = %q, want text", got[0].Type)
	}
	if got[0].Text != "Hello" {
		t.Errorf("Text = %q, want Hello", got[0].Text)
	}
}

func TestProjectNonClaudeJSON(t *testing.T) {
	// 未知的 JSON 事件应该被过滤（避免原始 JSON 泄露到前端）
	data := `{"key":"value"}`
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte(data)}}
	got := Project(in, "sess_test")
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0 (unknown JSON should be filtered)", len(got))
	}
}

func TestProjectRealStreamJSONToolUse(t *testing.T) {
	// 真实 Claude stream-json 格式：tool_use 作为 content_block 跨越多个事件。
	tracker := &PhaseTracker{
		pendingToolUses: make(map[int]*pendingToolUse),
		toolUseIDs:      make(map[string]string),
	}
	events := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte(`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_01","name":"Bash","input":{}}}`)},
		{Kind: engine.EventRaw, Data: []byte(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\":"}}`)},
		{Kind: engine.EventRaw, Data: []byte(`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"go test ./...\"}"}}`)},
		{Kind: engine.EventRaw, Data: []byte(`{"type":"content_block_stop","index":1}`)},
	}
	var got []Event
	for _, e := range events {
		ev, err := parseClaudeEventWithTracker(e.Data, "sess_test", tracker)
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		got = append(got, ev...)
	}
	// 期望：content_block_start 不 emit；2 个 delta 累积；content_block_stop 时统一 emit 一个 ToolUseEvent
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (only ToolUseEvent at stop), got %d events: %+v", len(got), len(got), eventTypes(got))
	}
	if got[0].Type != EventToolUse {
		t.Fatalf("Type = %q, want tool_use", got[0].Type)
	}
	if got[0].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash", got[0].ToolName)
	}
	if got[0].ToolID != "toolu_01" {
		t.Errorf("ToolID = %q, want toolu_01 (should preserve tool_use.id for pairing)", got[0].ToolID)
	}
	// ToolInput 应该是解析后的 map
	input, ok := got[0].ToolInput.(map[string]any)
	if !ok {
		t.Fatalf("ToolInput type = %T, want map[string]any", got[0].ToolInput)
	}
	if input["command"] != "go test ./..." {
		t.Errorf("ToolInput.command = %v, want \"go test ./...\"", input["command"])
	}
}

func TestProjectRealStreamJSONToolResult(t *testing.T) {
	// 真实 Claude 格式：tool_result 在 user 消息的 content 数组中
	tracker := &PhaseTracker{
		pendingToolUses: make(map[int]*pendingToolUse),
		toolUseIDs:      make(map[string]string),
	}
	// 先登记 tool_use_id → name 映射
	tracker.toolUseIDs["toolu_01"] = "Bash"
	data := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_01","content":"ok\n","is_error":false}]}}`
	ev, err := parseClaudeEventWithTracker([]byte(data), "sess_test", tracker)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ev) != 1 {
		t.Fatalf("len = %d, want 1, got %+v", len(ev), eventTypes(ev))
	}
	if ev[0].Type != EventToolResult {
		t.Fatalf("Type = %q, want tool_result", ev[0].Type)
	}
	if ev[0].ToolName != "Bash" {
		t.Errorf("ToolName = %q, want Bash (should be looked up from toolUseIDs)", ev[0].ToolName)
	}
}

func TestProjectRealStreamJSONFullSequence(t *testing.T) {
	// 完整序列：text → tool_use → tool_result → text → turn_end
	tracker := &PhaseTracker{
		pendingToolUses: make(map[int]*pendingToolUse),
		toolUseIDs:      make(map[string]string),
	}
	events := []string{
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Let me run tests."}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_42","name":"Bash","input":{}}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\"}"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_42","content":"file1\nfile2","is_error":false}]}}`,
		`{"type":"result","subtype":"success","is_error":false,"result":"Done","duration_ms":100}`,
	}
	var got []Event
	for _, data := range events {
		ev, err := parseClaudeEventWithTracker([]byte(data), "sess_test", tracker)
		if err != nil {
			t.Fatalf("parse %q: %v", data, err)
		}
		got = append(got, ev...)
	}
	types := eventTypes(got)
	// 期望事件类型序列：text_delta, tool_use, tool_result, turn_end
	wantTypes := []EventType{EventTextDelta, EventToolUse, EventToolResult, EventTurnEnd}
	if len(types) != len(wantTypes) {
		t.Fatalf("event count = %d, want %d. Got types: %v", len(types), len(wantTypes), types)
	}
	for i, wt := range wantTypes {
		if types[i] != wt {
			t.Errorf("event[%d].Type = %q, want %q (full sequence: %v)", i, types[i], wt, types)
		}
	}
	// 验证 tool_result 的 ToolName 正确
	toolResultEv := got[2]
	if toolResultEv.ToolName != "Bash" {
		t.Errorf("ToolResult.ToolName = %q, want Bash", toolResultEv.ToolName)
	}
}

func eventTypes(events []Event) []EventType {
	out := make([]EventType, 0, len(events))
	for _, e := range events {
		out = append(out, e.Type)
	}
	return out
}
