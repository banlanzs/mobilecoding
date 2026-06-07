package engine

import (
	"runtime"
	"testing"
)

func TestNativeRunnerUsesManagedClaudeRunnerOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Claude Code 交互式 CLI 不能运行在普通 stdin/stdout pipe 下")
	}

	r := NewNativeRunner("claude")
	if _, ok := r.(*ClaudeRunner); !ok {
		t.Fatalf("NewNativeRunner(claude) = %T, want *ClaudeRunner", r)
	}
}
