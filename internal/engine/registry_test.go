package engine

import "testing"

func TestRegistryReturnsPtyForUnknown(t *testing.T) {
	r, err := NewRunner("aichat", ExecRequest{})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, ok := r.(*PtyRunner); !ok {
		t.Errorf("unknown command should fall back to PtyRunner, got %T", r)
	}
	_ = r.Close()
}

func TestRegistryRejectsEmpty(t *testing.T) {
	if _, err := NewRunner("", ExecRequest{}); err == nil {
		t.Errorf("NewRunner(\"\") should fail")
	}
}
