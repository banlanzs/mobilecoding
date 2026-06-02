package engine

import (
	"testing"
)

func TestNewCodexRunner(t *testing.T) {
	r := NewCodexRunner()
	if r.SessionID() == "" {
		t.Fatal("SessionID should not be empty")
	}
	if r.Events() == nil {
		t.Fatal("Events channel should not be nil")
	}
	if r.Errors() == nil {
		t.Fatal("Errors channel should not be nil")
	}
	if r.Done() == nil {
		t.Fatal("Done channel should not be nil")
	}
}

func TestCodexRunnerImplementsInterface(t *testing.T) {
	var _ Runner = NewCodexRunner()
}

func TestCodexRunnerCanAcceptInteractiveInput(t *testing.T) {
	r := NewCodexRunner()
	if !r.CanAcceptInteractiveInput() {
		t.Error("CodexRunner should accept interactive input")
	}
}

func TestCodexRunnerHasActiveTurnBeforeStart(t *testing.T) {
	r := NewCodexRunner()
	if r.HasActiveTurn() {
		t.Error("HasActiveTurn should be false before Start")
	}
}
