package engine

import (
	"encoding/json"
	"testing"
)

func TestFormatClaudeInputWrapsTextAsStreamJSON(t *testing.T) {
	line, err := formatClaudeInput([]byte("你好"))
	if err != nil {
		t.Fatalf("formatClaudeInput: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(line, &got); err != nil {
		t.Fatalf("input line should be JSON: %v", err)
	}
	if got["type"] != "user_message" {
		t.Fatalf("type = %v, want user_message", got["type"])
	}
	if got["content"] != "你好" {
		t.Fatalf("content = %v, want 你好", got["content"])
	}
	if line[len(line)-1] != '\n' {
		t.Fatalf("input line should end with newline")
	}
}
