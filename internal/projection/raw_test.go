package projection

import (
	"testing"

	"github.com/jaycrl/mytool/internal/engine"
)

func TestProjectUsesProvidedSessionID(t *testing.T) {
	in := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte("hello\n")},
	}
	got := Project(in, "sess_fixed_42")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].SessionID != "sess_fixed_42" {
		t.Errorf("SessionID = %q, want sess_fixed_42", got[0].SessionID)
	}
}

func TestProjectFallsBackToUUID(t *testing.T) {
	in := []engine.Event{{Kind: engine.EventRaw, Data: []byte("hi")}}
	got := Project(in, "")
	if got[0].SessionID == "" {
		t.Error("Project should generate uuid when sid is empty")
	}
	if got[0].SessionID[:5] != "sess_" {
		t.Errorf("generated sid should start with sess_, got %q", got[0].SessionID)
	}
}

func TestTextEvent(t *testing.T) {
	e := TextEvent("s1", "hello")
	if e.Type != EventText {
		t.Errorf("Type = %q, want text", e.Type)
	}
	if e.SessionID != "s1" || e.Text != "hello" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestLifecycleEvent(t *testing.T) {
	e := LifecycleEvent("s1", "started")
	if e.Type != EventLifecycle {
		t.Errorf("Type = %q, want lifecycle", e.Type)
	}
	if e.SessionID != "s1" || e.Message != "started" {
		t.Errorf("unexpected: %+v", e)
	}
}

func TestStreamWithSID(t *testing.T) {
	in := make(chan engine.Event, 2)
	in <- engine.Event{Kind: engine.EventRaw, Data: []byte("a\n")}
	in <- engine.Event{Kind: engine.EventLifecycle, Message: "x"}
	close(in)
	out := make(chan Event, 4)
	Stream(in, out, "sess_stream")
	var got []Event
	for e := range out {
		got = append(got, e)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].SessionID != "sess_stream" || got[1].SessionID != "sess_stream" {
		t.Errorf("SessionIDs should both be sess_stream, got %q / %q", got[0].SessionID, got[1].SessionID)
	}
}
