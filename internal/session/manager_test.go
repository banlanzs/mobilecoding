package session

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	events  chan Event
	errors  chan error
	done    chan struct{}
	closed  bool
	mu      sync.Mutex
	started bool
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{
		events: make(chan Event, 8),
		errors: make(chan error, 1),
		done:   make(chan struct{}),
	}
}

func (f *fakeRunner) Start(ctx context.Context, req ExecRequest) error {
	f.mu.Lock()
	f.started = true
	f.mu.Unlock()
	go func() {
		// 推一条事件后退出
		f.events <- Event{Kind: "raw", Data: []byte("hello")}
		close(f.events)
		close(f.done)
	}()
	return nil
}
func (f *fakeRunner) Write(p []byte) error            { return nil }
func (f *fakeRunner) Resize(c, r int) error           { return nil }
func (f *fakeRunner) Close() error                    { f.mu.Lock(); defer f.mu.Unlock(); f.closed = true; return nil }
func (f *fakeRunner) Events() <-chan Event            { return f.events }
func (f *fakeRunner) Errors() <-chan error            { return f.errors }
func (f *fakeRunner) Done() <-chan struct{}           { return f.done }
func (f *fakeRunner) SessionID() string               { return "fake" }
func (f *fakeRunner) CanAcceptInteractiveInput() bool { return true }
func (f *fakeRunner) HasActiveTurn() bool             { return true }
func (f *fakeRunner) SendToStdin(p []byte) error     { return nil }
func (f *fakeRunner) Abort()                         {}

func TestManagerStartAndCollect(t *testing.T) {
	m := NewManager()
	run := newFakeRunner()
	sid, err := m.Start(context.Background(), ExecRequest{Command: "x"}, run)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if sid == "" {
		t.Errorf("session id should not be empty")
	}
	// 推一个事件
	run.events <- Event{Kind: "raw", Data: []byte("foo")}
	// 拿事件
	select {
	case ev := <-m.Output():
		if string(ev.Data) != "foo" {
			t.Errorf("output event data = %q, want foo", string(ev.Data))
		}
	case <-time.After(1 * time.Second):
		t.Errorf("timeout waiting for output event")
	}
	_ = m.Stop()
}

func TestManagerRejectsDoubleStart(t *testing.T) {
	m := NewManager()
	run1 := newFakeRunner()
	run2 := newFakeRunner()
	if _, err := m.Start(context.Background(), ExecRequest{Command: "x"}, run1); err != nil {
		t.Fatalf("Start 1: %v", err)
	}
	if _, err := m.Start(context.Background(), ExecRequest{Command: "x"}, run2); err == nil {
		t.Errorf("second Start should fail while first is active")
	}
	_ = m.Stop()
}

func TestManagerStop(t *testing.T) {
	m := NewManager()
	run := newFakeRunner()
	_, _ = m.Start(context.Background(), ExecRequest{Command: "x"}, run)
	if err := m.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
}
