package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	// Snapshot env so we can restore
	keys := []string{
		"MYTOOL_PORT",
		"MYTOOL_AUTH_TOKEN",
		"MYTOOL_WORKSPACE",
		"MYTOOL_MTLS",
		"MYTOOL_LOG_LEVEL",
		"MYTOOL_DEFAULT_COMMAND",
	}
	for _, k := range keys {
		orig, had := os.LookupEnv(k)
		if had {
			t.Cleanup(func() { _ = os.Setenv(k, orig) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv(k) })
		}
		_ = os.Unsetenv(k)
	}

	if got := FromEnv(); got.Port != "" || got.AuthToken != "" || got.Workspace != "" {
		t.Errorf("FromEnv() with empty env should return zero values, got %+v", got)
	}

	os.Setenv("MYTOOL_PORT", "9999")
	os.Setenv("MYTOOL_AUTH_TOKEN", "tok-abc")
	os.Setenv("MYTOOL_WORKSPACE", "/tmp/ws")
	os.Setenv("MYTOOL_MTLS", "required")
	os.Setenv("MYTOOL_LOG_LEVEL", "debug")
	os.Setenv("MYTOOL_DEFAULT_COMMAND", "codex")

	got := FromEnv()
	if got.Port != "9999" {
		t.Errorf("Port = %q, want 9999", got.Port)
	}
	if got.AuthToken != "tok-abc" {
		t.Errorf("AuthToken = %q, want tok-abc", got.AuthToken)
	}
	if got.Workspace != "/tmp/ws" {
		t.Errorf("Workspace = %q, want /tmp/ws", got.Workspace)
	}
	if got.MTLS != "required" {
		t.Errorf("MTLS = %q, want required", got.MTLS)
	}
	if got.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want debug", got.LogLevel)
	}
	if got.DefaultCmd != "codex" {
		t.Errorf("DefaultCmd = %q, want codex", got.DefaultCmd)
	}
}
