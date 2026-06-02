package engine

import (
	"testing"
)

func TestCodexTransportParse(t *testing.T) {
	line := `{"method":"thread/started","params":{"thread_id":"abc"}}`
	ev, err := ParseCodexJSONRPC([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	if len(ev.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestCodexTransportEmptyLine(t *testing.T) {
	ev, err := ParseCodexJSONRPC([]byte(""))
	if err != nil {
		t.Errorf("empty line should not error: %v", err)
	}
	if ev.Data != nil {
		t.Error("empty line should produce nil Data")
	}
}

func TestCodexTransportInvalidJSON(t *testing.T) {
	_, err := ParseCodexJSONRPC([]byte("not json"))
	if err == nil {
		t.Error("invalid JSON should error")
	}
}
