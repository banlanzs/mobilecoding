package engine

import (
	"runtime"
	"testing"
)

func TestNativeRunnerUsesManagedClaudeRunnerOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PipeRunner 才会触发 Claude 非 TTY stdin 退出问题")
	}

	r := NewNativeRunner("claude")
	if _, ok := r.(*ClaudeRunner); !ok {
		t.Fatalf("NewNativeRunner(claude) = %T, want *ClaudeRunner", r)
	}
}
