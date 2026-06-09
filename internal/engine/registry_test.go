package engine

import (
	"runtime"
	"testing"
)

func TestRegistryReturnsPtyForUnknown(t *testing.T) {
	r, err := NewRunner("aichat", ExecRequest{})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if runtime.GOOS == "windows" {
		if _, ok := r.(*PipeRunner); !ok {
			t.Errorf("unknown command on windows should fall back to PipeRunner, got %T", r)
		}
	} else {
		if _, ok := r.(*PtyRunner); !ok {
			t.Errorf("unknown command should fall back to PtyRunner, got %T", r)
		}
	}
	_ = r.Close()
}

func TestRegistryRejectsEmpty(t *testing.T) {
	if _, err := NewRunner("", ExecRequest{}); err == nil {
		t.Errorf("NewRunner(\"\") should fail")
	}
}

func TestNewNativeRunnerUsesManagedClaudeRunnerForClaudeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific regression: Claude Code exits under ordinary stdin/stdout pipes")
	}

	run := NewNativeRunner("claude")
	if _, ok := run.(*ClaudeRunner); !ok {
		t.Fatalf("NewNativeRunner(claude) = %T, want *ClaudeRunner on Windows", run)
	}
}
