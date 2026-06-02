package engine

import (
	"encoding/json"
	"testing"
)

func TestClaudeParserAssistantMessage(t *testing.T) {
	line := `{"type":"assistant_message","message":"Hello"}`
	ev, err := ParseClaudeStreamJSON([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	if len(ev.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestClaudeParserToolUse(t *testing.T) {
	line := `{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`
	ev, err := ParseClaudeStreamJSON([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	var m map[string]any
	if err := json.Unmarshal(ev.Data, &m); err != nil {
		t.Errorf("Data should be valid JSON: %v", err)
	}
}

func TestClaudeParserEmptyLine(t *testing.T) {
	ev, err := ParseClaudeStreamJSON([]byte(""))
	if err != nil {
		t.Errorf("empty line should not error: %v", err)
	}
	if ev.Data != nil {
		t.Error("empty line should produce nil Data")
	}
}

func TestClaudeParserInvalidJSON(t *testing.T) {
	_, err := ParseClaudeStreamJSON([]byte("not json"))
	if err == nil {
		t.Error("invalid JSON should error")
	}
}