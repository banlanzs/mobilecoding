package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/google/uuid"
)

// CodexRunner 启动 codex app-server，通过 JSON-RPC over stdin/stdout 交互。
type CodexRunner struct {
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

func NewCodexRunner() *CodexRunner {
	return &CodexRunner{
		events:    make(chan Event, 64),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		sessionID: "codex_" + uuid.NewString(),
	}
}

func (r *CodexRunner) SessionID() string              { return r.sessionID }
func (r *CodexRunner) Events() <-chan Event            { return r.events }
func (r *CodexRunner) Errors() <-chan error            { return r.errors }
func (r *CodexRunner) Done() <-chan struct{}           { return r.done }
func (r *CodexRunner) CanAcceptInteractiveInput() bool { return true }
func (r *CodexRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil && r.cmd.Process != nil
}

func (r *CodexRunner) Start(ctx context.Context, req ExecRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}
	args := append([]string{"app-server"}, req.Args...)
	cmd := exec.CommandContext(ctx, req.Command, args...)
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
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start codex: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: codex"}

	go r.readLoop(stdout)
	go r.waitLoop()
	return nil
}

func (r *CodexRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.stdin == nil {
		return errors.New("runner is closed")
	}
	_, err := r.stdin.Write(p)
	return err
}

func (r *CodexRunner) SendToStdin(p []byte) error { return r.Write(p) }
func (r *CodexRunner) Abort()                       {}

func (r *CodexRunner) Resize(cols, rows int) error {
	return nil
}

func (r *CodexRunner) readLoop(stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, err := ParseCodexJSONRPC(line)
		if err != nil {
			r.errors <- fmt.Errorf("codex parse: %w", err)
			continue
		}
		select {
		case r.events <- ev:
		default:
			select {
			case r.errors <- errors.New("events channel full, dropping chunk"):
			default:
			}
		}
	}
}

func (r *CodexRunner) waitLoop() {
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

func (r *CodexRunner) Close() error {
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
