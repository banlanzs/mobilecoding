package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

// PipeRunner 是 PtyRunner 在 Windows 上的回退实现，使用 stdin/stdout 管道
// 而非伪终端。适用于不支持 creack/pty 的平台。
type PipeRunner struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	events    chan Event
	errors    chan error
	done      chan struct{}
	sessionID string
	mu        sync.Mutex
	closed    bool
}

func NewPipeRunner() *PipeRunner {
	return &PipeRunner{
		events:    make(chan Event, 64),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		sessionID: "pipe_" + uuid.NewString(),
	}
}

func (r *PipeRunner) SessionID() string               { return r.sessionID }
func (r *PipeRunner) Events() <-chan Event            { return r.events }
func (r *PipeRunner) Errors() <-chan error            { return r.errors }
func (r *PipeRunner) Done() <-chan struct{}           { return r.done }
func (r *PipeRunner) CanAcceptInteractiveInput() bool { return true }
func (r *PipeRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil && r.cmd.Process != nil
}

func (r *PipeRunner) Start(ctx context.Context, req ExecRequest) error {
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", req.Command, err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: " + req.Command}

	go r.readLoop(stdout)
	go r.readLoop(stderr)
	go r.waitLoop()
	return nil
}

func (r *PipeRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.stdin == nil {
		return errors.New("runner is closed")
	}
	_, err := r.stdin.Write(p)
	return err
}

func (r *PipeRunner) SendToStdin(p []byte) error { return r.Write(p) }
func (r *PipeRunner) Abort()                     {}

func (r *PipeRunner) Resize(_, _ int) error {
	return nil
}

func (r *PipeRunner) readLoop(reader io.ReadCloser) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case r.events <- Event{Kind: EventRaw, Data: data}:
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

func (r *PipeRunner) waitLoop() {
	err := r.cmd.Wait()
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	if err != nil {
		r.errors <- err
		r.events <- Event{Kind: EventLifecycle, Message: "exited: " + err.Error()}
	} else {
		r.events <- Event{Kind: EventLifecycle, Message: "exited"}
	}
	close(r.events)
	close(r.errors)
	close(r.done)
}

func (r *PipeRunner) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	if r.stdin != nil {
		r.stdin.Close()
	}
	cmd := r.cmd
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
