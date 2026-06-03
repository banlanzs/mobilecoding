package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

type PtyRunner struct {
	cmd       *exec.Cmd
	ptyFile   *os.File
	events    chan Event
	errors    chan error
	done      chan struct{}
	sessionID string
	mu        sync.Mutex
	closed    bool
	canInput  bool
}

func NewPtyRunner() *PtyRunner {
	return &PtyRunner{
		events:    make(chan Event, 64),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		canInput:  true,
		sessionID: "pty_" + uuid.NewString(),
	}
}

func (r *PtyRunner) SessionID() string     { return r.sessionID }
func (r *PtyRunner) Events() <-chan Event  { return r.events }
func (r *PtyRunner) Errors() <-chan error  { return r.errors }
func (r *PtyRunner) Done() <-chan struct{} { return r.done }

func (r *PtyRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.ptyFile == nil {
		return errors.New("runner is closed")
	}
	_, err := r.ptyFile.Write(p)
	return err
}

func (r *PtyRunner) Resize(cols, rows int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ptyFile == nil {
		return errors.New("pty not started")
	}
	return pty.Setsize(r.ptyFile, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func (r *PtyRunner) CanAcceptInteractiveInput() bool { return r.canInput }

func (r *PtyRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil && r.cmd.Process != nil
}

func (r *PtyRunner) Start(ctx context.Context, req ExecRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}
	cmd := exec.CommandContext(ctx, req.Command, req.Args...)
	if req.CWD != "" {
		cmd.Dir = req.CWD
	}
	if len(req.Env) > 0 {
		cmd.Env = append(os.Environ(), req.Env...)
	}
	cols, rows := req.Cols, req.Rows
	if cols == 0 {
		cols = 120
	}
	if rows == 0 {
		rows = 32
	}
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		return fmt.Errorf("pty start: %w", err)
	}
	r.mu.Lock()
	r.cmd = cmd
	r.ptyFile = ptmx
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: " + req.Command}

	go r.readLoop(ptmx)
	go r.waitLoop()
	return nil
}

func (r *PtyRunner) readLoop(f *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			cp := make([]byte, n)
			copy(cp, buf[:n])
			select {
			case r.events <- Event{Kind: EventRaw, Data: cp}:
			default:
				select {
				case r.errors <- errors.New("events channel full, dropping chunk"):
				default:
				}
			}
		}
		if err != nil {
			return
		}
	}
}

func (r *PtyRunner) waitLoop() {
	err := r.cmd.Wait()
	if err != nil {
		select {
		case r.errors <- err:
		default:
		}
	}
	msg := "exited"
	if err != nil {
		msg = "exited: " + err.Error()
	}
	select {
	case r.events <- Event{Kind: EventLifecycle, Message: msg}:
	default:
		select {
		case r.errors <- errors.New("events channel full, dropping lifecycle event"):
		default:
		}
	}
	// 等待 readLoop 读完所有输出后再关 channels
	time.Sleep(200 * time.Millisecond)
	close(r.events)
	close(r.errors)
	close(r.done)
}

func (r *PtyRunner) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	ptmx := r.ptyFile
	cmd := r.cmd
	r.mu.Unlock()
	if ptmx != nil {
		_ = ptmx.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
