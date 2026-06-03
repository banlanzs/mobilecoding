package engine

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// ClaudeRunner 启动 claude --print --output-format stream-json --input-format stream-json，
// 采用 lazy start：Start() 不启动进程，首条消息到达时才启动并写入 stdin。
type ClaudeRunner struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	events    chan Event
	errors    chan error
	done      chan struct{}
	sessionID string
	mu        sync.Mutex
	closed    bool
	started   bool
	ctx       context.Context
	req       ExecRequest
}

func NewClaudeRunner() *ClaudeRunner {
	return &ClaudeRunner{
		events:    make(chan Event, 64),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		sessionID: "claude_" + uuid.NewString(),
	}
}

func (r *ClaudeRunner) SessionID() string              { return r.sessionID }
func (r *ClaudeRunner) Events() <-chan Event            { return r.events }
func (r *ClaudeRunner) Errors() <-chan error            { return r.errors }
func (r *ClaudeRunner) Done() <-chan struct{}           { return r.done }
func (r *ClaudeRunner) CanAcceptInteractiveInput() bool { return true }
func (r *ClaudeRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started && r.cmd != nil && r.cmd.Process != nil
}

// Start 保存配置但不启动进程（lazy start）。
// 首条 Write() 到达时才真正启动 claude。
func (r *ClaudeRunner) Start(ctx context.Context, req ExecRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}
	r.ctx = ctx
	r.req = req
	r.events <- Event{Kind: EventLifecycle, Message: "ready: claude (waiting for first message)"}
	return nil
}

// startProcess 启动 claude 子进程并立即写入首条消息。
func (r *ClaudeRunner) startProcess(firstInput []byte) error {
	// Windows 上需要使用 cmd /c 来启动 npm 安装的命令
	command := r.req.Command
	var args []string
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(command), ".exe") {
		args = []string{"/c", command, "--print", "--verbose", "--output-format", "stream-json", "--input-format", "stream-json", "--permission-prompt-tool", "stdio"}
		args = append(args, r.req.Args...)
		command = "cmd"
	} else {
		args = append([]string{"--print", "--verbose", "--output-format", "stream-json", "--input-format", "stream-json", "--permission-prompt-tool", "stdio"}, r.req.Args...)
	}

	cmd := exec.CommandContext(r.ctx, command, args...)
	if r.req.CWD != "" {
		cmd.Dir = r.req.CWD
	}
	if len(r.req.Env) > 0 {
		cmd.Env = append(os.Environ(), r.req.Env...)
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
		return fmt.Errorf("start claude: %w", err)
	}

	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.started = true

	// 立即写入首条消息
	inputLine, err := formatClaudeInput(firstInput)
	if err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("format first input: %w", err)
	}
	if _, err := stdin.Write(inputLine); err != nil {
		_ = cmd.Process.Kill()
		return fmt.Errorf("write first input: %w", err)
	}

	r.events <- Event{Kind: EventLifecycle, Message: "started: claude"}

	go r.readLoop(stdout)
	go r.readStderr(stderr)
	go r.waitLoop()
	return nil
}

// Write 写入用户消息。首次调用时启动进程。
func (r *ClaudeRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return errors.New("runner is closed")
	}
	if !r.started {
		return r.startProcess(p)
	}
	line, err := formatClaudeInput(p)
	if err != nil {
		return err
	}
	_, err = r.stdin.Write(line)
	return err
}

func formatClaudeInput(p []byte) ([]byte, error) {
	content := strings.TrimRight(string(p), "\r\n")
	msg := map[string]any{
		"type":    "user_message",
		"content": content,
	}
	line, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal claude input: %w", err)
	}
	return append(line, '\n'), nil
}

func (r *ClaudeRunner) Resize(cols, rows int) error {
	return nil
}

func (r *ClaudeRunner) readLoop(stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		ev, err := ParseClaudeStreamJSON(line)
		if err != nil {
			r.errors <- fmt.Errorf("claude parse: %w", err)
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

func (r *ClaudeRunner) readStderr(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		select {
		case r.errors <- fmt.Errorf("claude stderr: %s", line):
		default:
		}
	}
}

func (r *ClaudeRunner) waitLoop() {
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

func (r *ClaudeRunner) Close() error {
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
