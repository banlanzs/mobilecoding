package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayClaudeSettingsUsesDefaultLabelWhenEmpty(t *testing.T) {
	if got := displayClaudeSettings(""); got != "默认配置" {
		t.Fatalf("displayClaudeSettings(empty) = %q, want 默认配置", got)
	}
}

func TestDisplayClaudeSettingsReturnsProvidedPath(t *testing.T) {
	path := `C:\Users\banlan\.claude\settings.axonhub.json`
	if got := displayClaudeSettings(path); got != path {
		t.Fatalf("displayClaudeSettings(path) = %q, want %q", got, path)
	}
}

func TestDisplayClaudeModelUsesDefaultLabelWhenEmpty(t *testing.T) {
	if got := displayClaudeModel(""); got != "默认模型" {
		t.Fatalf("displayClaudeModel(empty) = %q, want 默认模型", got)
	}
}

func TestDisplayClaudeModelReturnsProvidedModel(t *testing.T) {
	model := "claude-opus-4-8"
	if got := displayClaudeModel(model); got != model {
		t.Fatalf("displayClaudeModel(model) = %q, want %q", got, model)
	}
}

func TestDetectProjectSettingsInFindsLocalSettings(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(claudeDir, "settings.local.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := detectProjectSettingsIn(dir)
	if got != target {
		t.Fatalf("detectProjectSettingsIn = %q, want %q", got, target)
	}
}

func TestDetectProjectSettingsInReturnsEmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if got := detectProjectSettingsIn(dir); got != "" {
		t.Fatalf("detectProjectSettingsIn(missing) = %q, want empty", got)
	}
}

func TestDetectProjectSettingsInIgnoresDirectory(t *testing.T) {
	dir := t.TempDir()
	// settings.local.json 是目录而非文件，应被忽略
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "settings.local.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := detectProjectSettingsIn(dir); got != "" {
		t.Fatalf("detectProjectSettingsIn(dir-as-file) = %q, want empty", got)
	}
}

