package ws

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeReqRoundTrip(t *testing.T) {
	in := Envelope{
		Type:   "req",
		ID:     "u-1",
		Method: "session.start",
		Params: json.RawMessage(`{"command":"echo","args":["hi"]}`),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Envelope
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != "req" || out.ID != "u-1" || out.Method != "session.start" {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestEnvelopeRespError(t *testing.T) {
	raw := `{"type":"resp","id":"u-2","ok":false,"error":{"code":"unauthorized","message":"bad token"}}`
	var e Envelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Error == nil || e.Error.Code != "unauthorized" {
		t.Errorf("error decode wrong: %+v", e.Error)
	}
}

func TestEnvelopeEvt(t *testing.T) {
	raw := `{"type":"evt","sessionId":"sess_1","event":{"type":"text","text":"hello","sessionId":"sess_1"}}`
	var e Envelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != "evt" || e.Event == nil {
		t.Errorf("event decode wrong: %+v", e)
	}
}
