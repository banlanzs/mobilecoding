package engine

import (
	"encoding/json"
	"reflect"
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

func TestClaudeProcessArgsPreservesSettingsFlag(t *testing.T) {
	got := claudeProcessArgs([]string{"--settings", `C:\Users\banlan\.claude\settings.ccload.json`, "--model", "claude-opus-4-8"}, "session-123")
	want := []string{
		"--print",
		"--verbose",
		"--output-format", "stream-json",
		"--input-format", "stream-json",
		"--permission-prompt-tool", "stdio",
		"--resume", "session-123",
		"--settings", `C:\Users\banlan\.claude\settings.ccload.json`,
		"--model", "claude-opus-4-8",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("claudeProcessArgs() = %#v, want %#v", got, want)
	}
}

func TestClaudeProcessEnvClearsInheritedProviderEnvWhenSettingsProvided(t *testing.T) {
	base := []string{
		"PATH=/bin",
		"ANTHROPIC_BASE_URL=https://current-provider.example",
		"ANTHROPIC_AUTH_TOKEN=current-token",
		"ANTHROPIC_MODEL=current-model",
		"CLAUDE_CODE_USE_VERTEX=1",
		"CLAUDE_CODE_OAUTH_TOKEN=current-oauth",
		"MOBILECODING_TOKEN=keep",
	}
	got := claudeProcessEnv(base, []string{"ANTHROPIC_BASE_URL=https://explicit.example", "CUSTOM=1"}, true)
	want := []string{
		"PATH=/bin",
		"MOBILECODING_TOKEN=keep",
		"ANTHROPIC_BASE_URL=https://explicit.example",
		"CUSTOM=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("claudeProcessEnv(settings) = %#v, want %#v", got, want)
	}
}

func TestClaudeProcessEnvPreservesInheritedProviderEnvWithoutSettings(t *testing.T) {
	base := []string{"PATH=/bin", "ANTHROPIC_BASE_URL=https://current-provider.example"}
	got := claudeProcessEnv(base, nil, false)
	if !reflect.DeepEqual(got, base) {
		t.Fatalf("claudeProcessEnv(no settings) = %#v, want %#v", got, base)
	}
}
