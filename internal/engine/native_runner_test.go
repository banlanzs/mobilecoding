package engine

import (
	"runtime"
	"testing"
)

func TestNativeRunnerUsesInteractivePipeRunnerOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows remote-control 才需要 PipeRunner 交互式 stdin/stdout")
	}

	r := NewNativeRunner("claude")
	if _, ok := r.(*PipeRunner); !ok {
		t.Fatalf("NewNativeRunner(claude) = %T, want *PipeRunner", r)
	}
}
