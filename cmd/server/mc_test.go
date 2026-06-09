package main

import "testing"

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
