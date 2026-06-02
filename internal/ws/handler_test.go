package ws

import (
	"encoding/json"
	"testing"

	"github.com/banlanzs/mobilecoding/internal/session"
)

func TestNewErrorResp(t *testing.T) {
	e := newErrorResp("u-1", "test_code", "test msg")
	if e.Type != "resp" {
		t.Errorf("Type = %q, want resp", e.Type)
	}
	if e.ID != "u-1" {
		t.Errorf("ID = %q, want u-1", e.ID)
	}
	if e.OK == nil || *e.OK != false {
		t.Errorf("OK should be &false, got %v", e.OK)
	}
	if e.Error == nil || e.Error.Code != "test_code" {
		t.Errorf("Error.Code = %v, want test_code", e.Error)
	}
}

func TestNewErrorRespPtr(t *testing.T) {
	e := newErrorRespPtr("u-2", "code2", "msg2")
	if e == nil {
		t.Fatal("expected non-nil pointer")
	}
	if e.ID != "u-2" || e.Error.Code != "code2" {
		t.Errorf("got %+v", e)
	}
}

func TestDispatchUnknownMethod(t *testing.T) {
	h := NewHandler(NewHub(), session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{Type: "req", ID: "u-x", Method: "session.bogus"})
	if resp == nil {
		t.Fatal("expected non-nil resp")
	}
	if resp.Error == nil || resp.Error.Code != "not_found" {
		t.Errorf("Error.Code = %v, want not_found", resp.Error)
	}
}

func TestDispatchStartInvalidParams(t *testing.T) {
	h := NewHandler(NewHub(), session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{
		Type:   "req",
		ID:     "u-y",
		Method: "session.start",
		Params: json.RawMessage(`{"command":`), // malformed
	})
	if resp == nil || resp.Error == nil || resp.Error.Code != "protocol_error" {
		t.Errorf("expected protocol_error, got %+v", resp)
	}
}

func TestDispatchStopNoActive(t *testing.T) {
	h := NewHandler(NewHub(), session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{Type: "req", ID: "u-z", Method: "session.stop"})
	if resp == nil {
		t.Fatal("expected non-nil resp")
	}
	if resp.Error != nil {
		t.Errorf("Stop with no active runner should succeed (return ok=true), got error %+v", resp.Error)
	}
	if resp.OK == nil || *resp.OK != true {
		t.Errorf("OK should be &true, got %v", resp.OK)
	}
}

func TestDispatchInputNoActive(t *testing.T) {
	h := NewHandler(NewHub(), session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{
		Type:   "req",
		ID:     "u-w",
		Method: "session.input",
		Params: json.RawMessage(`{"text":"hi"}`),
	})
	if resp == nil || resp.Error == nil || resp.Error.Code != "engine_failure" {
		t.Errorf("expected engine_failure, got %+v", resp)
	}
}
