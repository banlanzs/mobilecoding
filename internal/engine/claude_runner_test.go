package engine

import (
	"encoding/json"
	"strings"
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
	// Claude --input-format stream-json 期望的格式：{"type":"user","message":{"role":"user","content":"..."}}
	if got["type"] != "user" {
		t.Fatalf("type = %v, want user", got["type"])
	}
	msg, ok := got["message"].(map[string]any)
	if !ok {
		t.Fatalf("message field missing or wrong type: %v", got["message"])
	}
	if msg["role"] != "user" {
		t.Fatalf("message.role = %v, want user", msg["role"])
	}
	if msg["content"] != "你好" {
		t.Fatalf("message.content = %v, want 你好", msg["content"])
	}
	if line[len(line)-1] != '\n' {
		t.Fatalf("input line should end with newline")
	}
}

func TestFormatClaudeInputPreservesNewlines(t *testing.T) {
	// 多行消息必须保留 \n，且整个消息必须是有效 JSON（避免 Windows CreateProcess 截断）
	multi := "Run go test ./...\n然后 ls 看看"
	line, err := formatClaudeInput([]byte(multi))
	if err != nil {
		t.Fatalf("formatClaudeInput: %v", err)
	}
	if !strings.Contains(string(line), "Run go test ./...") {
		t.Fatalf("line missing first line: %s", line)
	}
	if !strings.Contains(string(line), "ls 看看") {
		t.Fatalf("line missing second line: %s", line)
	}
	// 必须能反序列化回原文（JSON 转义保证 \n 不被截断）
	var got map[string]any
	if err := json.Unmarshal(line, &got); err != nil {
		t.Fatalf("invalid json: %v / line=%s", err, line)
	}
	msg := got["message"].(map[string]any)
	if msg["content"] != multi {
		t.Fatalf("content = %q, want %q", msg["content"], multi)
	}
}
