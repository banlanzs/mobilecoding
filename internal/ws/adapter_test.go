package ws

import (
	"encoding/json"
	"testing"

	"github.com/banlanzs/mobilecoding/internal/projection"
)

func TestProjectionToEnvelope(t *testing.T) {
	env, err := projectionToEnvelope(projection.TextEvent("sess_1", "hi"))
	if err != nil {
		t.Fatalf("projectionToEnvelope: %v", err)
	}
	if env.Type != "evt" {
		t.Errorf("Type = %q, want evt", env.Type)
	}
	if env.SessionID != "sess_1" {
		t.Errorf("SessionID = %q, want sess_1", env.SessionID)
	}
	if env.Event == nil {
		t.Fatal("Event should not be nil")
	}
	var got projection.Event
	if err := json.Unmarshal(env.Event, &got); err != nil {
		t.Fatalf("unmarshal Event: %v", err)
	}
	if got.Type != projection.EventText || got.Text != "hi" {
		t.Errorf("unmarshaled event wrong: %+v", got)
	}
}

func TestProjectionToEnvelope_Lifecycle(t *testing.T) {
	env, _ := projectionToEnvelope(projection.LifecycleEvent("s1", "exited"))
	var got projection.Event
	_ = json.Unmarshal(env.Event, &got)
	if got.Type != projection.EventLifecycle || got.Message != "exited" {
		t.Errorf("lifecycle event roundtrip wrong: %+v", got)
	}
}
