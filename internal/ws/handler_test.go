package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/banlanzs/mobilecoding/internal/engine"
	"github.com/banlanzs/mobilecoding/internal/logx"
	"github.com/banlanzs/mobilecoding/internal/projection"
	"github.com/banlanzs/mobilecoding/internal/session"
)

func newTestHandler(mgr *session.Manager) *Handler {
	return NewHandler(NewHub(), mgr, logx.New())
}

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
	h := newTestHandler(session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{Type: "req", ID: "u-x", Method: "session.bogus"})
	if resp == nil {
		t.Fatal("expected non-nil resp")
	}
	if resp.Error == nil || resp.Error.Code != "not_found" {
		t.Errorf("Error.Code = %v, want not_found", resp.Error)
	}
}

func TestDispatchStartInvalidParams(t *testing.T) {
	h := newTestHandler(session.NewManager())
	resp, _ := h.dispatch(nil, Envelope{
		Type:   "req",
		ID:     "u-y",
		Method: "session.start",
		Params: json.RawMessage(`{"command":`),
	})
	if resp == nil || resp.Error == nil || resp.Error.Code != "protocol_error" {
		t.Errorf("expected protocol_error, got %+v", resp)
	}
}

func TestDispatchStopNoActive(t *testing.T) {
	h := newTestHandler(session.NewManager())
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
	h := newTestHandler(session.NewManager())
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

type mockRunner struct {
	events    chan engine.Event
	errs      chan error
	done      chan struct{}
	closed    bool
	lastWrite []byte
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		events: make(chan engine.Event, 32),
		errs:   make(chan error, 8),
		done:   make(chan struct{}),
	}
}
func (r *mockRunner) Start(_ context.Context, _ engine.ExecRequest) error { return nil }
func (r *mockRunner) Write(p []byte) error {
	r.lastWrite = append([]byte{}, p...)
	return nil
}
func (r *mockRunner) Resize(_, _ int) error { return nil }
func (r *mockRunner) Close() error {
	r.closed = true
	close(r.done)
	return nil
}
func (r *mockRunner) Events() <-chan engine.Event           { return r.events }
func (r *mockRunner) Errors() <-chan error                  { return r.errs }
func (r *mockRunner) Done() <-chan struct{}                 { return r.done }
func (r *mockRunner) SessionID() string                     { return "mock-session" }
func (r *mockRunner) CanAcceptInteractiveInput() bool       { return false }
func (r *mockRunner) HasActiveTurn() bool                   { return true }
func (r *mockRunner) SendToStdin(p []byte) error            { return nil }
func (r *mockRunner) Abort()                                {}

func TestForwardSessionForwardsEvents(t *testing.T) {
	mgr := session.NewManager()
	h := newTestHandler(mgr)
	run := newMockRunner()

	sid, err := mgr.Start(context.Background(), engine.ExecRequest{Command: "echo"}, run)
	if err != nil {
		t.Fatalf("mgr.Start: %v", err)
	}
	if sid == "" {
		t.Fatal("expected non-empty session ID")
	}

	out := make(chan Envelope, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.forwardSession(ctx, out)
	run.events <- engine.Event{Kind: engine.EventRaw, Data: []byte("hello")}

	select {
	case env := <-out:
		if env.Type != "evt" {
			t.Errorf("type = %q, want evt", env.Type)
		}
		if env.SessionID == "" {
			t.Error("sessionId should not be empty")
		}
		var got projection.Event
		if err := json.Unmarshal(env.Event, &got); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if got.Type != "text" || got.Text != "hello" {
			t.Errorf("unexpected event: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for forwarded event")
	}
}

func TestForwardSessionContextCancel(t *testing.T) {
	mgr := session.NewManager()
	h := newTestHandler(mgr)

	out := make(chan Envelope, 16)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.forwardSession(ctx, out)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("forwardSession did not exit after context cancel")
	}
}

func TestHandleInputAppendsNewline(t *testing.T) {
	mgr := session.NewManager()
	h := newTestHandler(mgr)
	run := newMockRunner()

	_, err := mgr.Start(context.Background(), engine.ExecRequest{Command: "echo"}, run)
	if err != nil {
		t.Fatalf("mgr.Start: %v", err)
	}

	resp, _ := h.dispatch(nil, Envelope{
		Type:   "req",
		ID:     "u-input",
		Method: "session.input",
		Params: json.RawMessage(`{"text":"hi"}`),
	})
	if resp == nil || resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp)
	}
	if string(run.lastWrite) != "hi\n" {
		t.Fatalf("lastWrite = %q, want %q", string(run.lastWrite), "hi\\n")
	}
}