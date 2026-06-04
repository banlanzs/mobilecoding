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
	"time"

	"github.com/google/uuid"
)

// ClaudeRunner 启动 claude --print ... "message"，
// 每条消息启动新进程，通过 --resume 保持多轮对话上下文。
// stdin 管道用于权限应答等中间交互。
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
	req       ExecRequest
	logWindow *LogWindow

	resumeSessionID string // Claude 内部 session id，用于 --resume
	currentStdin    io.WriteCloser // 当前运行进程的 stdin（用于权限应答）
	wg              sync.WaitGroup // 追踪活跃 goroutines
}

func NewClaudeRunner() *ClaudeRunner {
	return &ClaudeRunner{
		events:    make(chan Event, 256),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		sessionID: "claude_" + uuid.NewString(),
	}
}

func (r *ClaudeRunner) SessionID() string              { return r.sessionID }
func (r *ClaudeRunner) Events() <-chan Event            { return r.events }
func (r *ClaudeRunner) Errors() <-chan error            { return r.errors }
func (r *ClaudeRunner) Done() <-chan struct{}           { return r.done }
func (r *ClaudeRunner) CanAcceptInteractiveInput() bool { return !r.closed }
func (r *ClaudeRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started && r.cmd != nil && r.cmd.Process != nil
}

func (r *ClaudeRunner) Start(ctx context.Context, req ExecRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}
	r.req = req
	r.events <- Event{Kind: EventLifecycle, Message: "ready: claude (waiting for first message)"}
	return nil
}

// runClaude 启动 claude --print ... "message"，一条消息一个进程。
// 多轮对话通过 --resume <session_id> 保持上下文。
func (r *ClaudeRunner) runClaude(prompt string) error {
	settingsEnv := extractSettingsEnv(r.req.Args)
	filteredArgs := filterSettingsArgs(r.req.Args)

	args := []string{"--print", "--verbose", "--output-format", "stream-json", "--permission-prompt-tool", "stdio"}

	// 多轮对话：续接上次 session
	r.mu.Lock()
	sid := r.resumeSessionID
	r.mu.Unlock()
	if sid != "" {
		args = append(args, "--resume", sid)
	}

	args = append(args, filteredArgs...)
	if prompt != "" {
		args = append(args, prompt)
	}

	command := r.req.Command
	if runtime.GOOS == "windows" {
		command = resolveWindowsCommand(command)
	}
	cmd := exec.CommandContext(context.Background(), command, args...)
	if r.req.CWD != "" {
		cmd.Dir = r.req.CWD
	}
	cmd.Env = os.Environ()
	for k, v := range settingsEnv {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	if len(r.req.Env) > 0 {
		cmd.Env = append(cmd.Env, r.req.Env...)
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

	r.mu.Lock()
	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.currentStdin = stdin
	r.started = true
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: claude"}

	r.wg.Add(3)
	go r.readLoop(stdout)
	go r.readStderr(stderr)
	go r.waitLoop()
	return nil
}

// Write 写入用户消息（启动新进程）。
func (r *ClaudeRunner) Write(p []byte) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return errors.New("runner is closed")
	}
	r.mu.Unlock()

	content := strings.TrimRight(string(p), "\r\n")
	if content == "" {
		return nil
	}

	if r.logWindow != nil {
		fmt.Fprintf(r.logWindow, "[USER INPUT] %s\n", content)
	}

	// 如果旧进程还在运行，先杀掉
	r.killProcess()
	r.mu.Lock()
	r.started = false
	r.mu.Unlock()

	return r.runClaude(content)
}

// Abort 中止当前请求：杀进程但不关 channels，session 可继续使用。
func (r *ClaudeRunner) Abort() {
	r.killProcess()
	r.mu.Lock()
	r.currentStdin = nil
	r.started = false
	r.mu.Unlock()
}

func (r *ClaudeRunner) killProcess() {
	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}
}


// SendToStdin 写入当前运行进程的 stdin（不杀进程）。
// 用于权限应答、--permission-prompt-tool stdio 等中间交互。
func (r *ClaudeRunner) SendToStdin(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.currentStdin == nil {
		return errors.New("no active process stdin")
	}
	if r.closed {
		return errors.New("runner is closed")
	}
	_, err := r.currentStdin.Write(p)
	return err
}

func (r *ClaudeRunner) Resize(cols, rows int) error {
	return nil
}

func (r *ClaudeRunner) readLoop(stdout io.ReadCloser) {
	defer r.wg.Done()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		r.captureResumeID(line)
		if r.logWindow != nil {
			fmt.Fprint(r.logWindow, formatStreamJSONForLog(line))
		}

		ev, err := ParseClaudeStreamJSON(line)
		if err != nil {
			select {
			case r.errors <- fmt.Errorf("claude parse: %w", err):
			case <-time.After(5 * time.Second):
			}
			continue
		}
		select {
		case r.events <- ev:
		case <-time.After(5 * time.Second):
			select {
			case r.errors <- fmt.Errorf("events channel blocked for 5s"):
			default:
			}
		}
	}
}

// captureResumeID 从事件行中提取 session_id 用于后续 --resume。
func (r *ClaudeRunner) captureResumeID(line []byte) {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return
	}
	if sid, ok := m["session_id"].(string); ok && sid != "" {
		r.mu.Lock()
		r.resumeSessionID = sid
		r.mu.Unlock()
	}
}

func (r *ClaudeRunner) readStderr(stderr io.ReadCloser) {
	defer r.wg.Done()
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if r.logWindow != nil {
			fmt.Fprintf(r.logWindow, "[STDERR] %s\n", line)
		}
		select {
		case r.errors <- fmt.Errorf("claude stderr: %s", line):
		default:
		}
	}
}

func (r *ClaudeRunner) waitLoop() {
	defer r.wg.Done()
	err := r.cmd.Wait()
	// 非阻塞发送，避免 Close() 关闭 channels 后 panic
	if err != nil {
		select {
		case r.errors <- err:
		default:
		}
		select {
		case r.events <- Event{Kind: EventLifecycle, Message: "turn complete: " + err.Error()}:
		default:
		}
	} else {
		select {
		case r.events <- Event{Kind: EventLifecycle, Message: "turn complete"}:
		default:
		}
	}
}

func (r *ClaudeRunner) Close() error {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	r.killProcess()
	// 等待所有 goroutines 结束，再关 channels
	r.wg.Wait()
	if r.logWindow != nil {
		r.logWindow.Close()
	}
	close(r.events)
	close(r.errors)
	close(r.done)
	return nil
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

// extractSettingsEnv 从 --settings 参数中读取 settings 文件并提取环境变量。
// resolveWindowsCommand 在 Windows 上找到可执行文件路径。
// 避免使用 cmd /c 导致 stdin 管道缓冲问题。
func resolveWindowsCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	if strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".cmd") || strings.HasSuffix(lower, ".bat") {
		return cmd
	}
	// 尝试找 claude.cmd / codex.cmd 等
	for _, ext := range []string{".cmd", ".exe", ".bat"} {
		if resolved, err := exec.LookPath(cmd + ext); err == nil {
			return resolved
		}
	}
	// 尝试直接查找
	if resolved, err := exec.LookPath(cmd); err == nil && resolved != "" {
		return resolved
	}
	return cmd
}

func extractSettingsEnv(args []string) map[string]string {
	settingsPath := ""
	for i, arg := range args {
		if arg == "--settings" && i+1 < len(args) {
			settingsPath = args[i+1]
			break
		}
	}
	if settingsPath == "" {
		return nil
	}

	settingsPath = os.ExpandEnv(settingsPath)
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	var settings struct {
		Env map[string]string `json:"env"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil
	}

	for k, v := range settings.Env {
		settings.Env[k] = os.ExpandEnv(v)
	}
	return settings.Env
}

// filterSettingsArgs 从参数列表中移除 --settings 参数。
func filterSettingsArgs(args []string) []string {
	var result []string
	skip := false
	for _, arg := range args {
		if skip {
			skip = false
			continue
		}
		if arg == "--settings" {
			skip = true
			continue
		}
		result = append(result, arg)
	}
	return result
}

// formatStreamJSONForLog 将 Claude stream-json 解析为可读文本。
func formatStreamJSONForLog(line []byte) string {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return fmt.Sprintf("[RAW] %s\n", string(line))
	}

	typ, _ := m["type"].(string)
	switch typ {
	case "system":
		subtype, _ := m["subtype"].(string)
		if subtype == "hook_started" || subtype == "hook_response" {
			return ""
		}
		return fmt.Sprintf("[SYSTEM] %s\n", subtype)
	case "assistant", "assistant_message":
		msg := m["message"]
		text := extractContentBlocks(msg)
		if text == "" {
			return ""
		}
		return fmt.Sprintf("\n🤖 Claude:\n%s\n", text)
	case "tool_use":
		name, _ := m["name"].(string)
		return fmt.Sprintf("🔧 Tool: %s\n", name)
	case "tool_result":
		name, _ := m["name"].(string)
		return fmt.Sprintf("✅ Tool Result: %s\n", name)
	case "permission_request":
		toolName, _ := m["tool_name"].(string)
		prompt, _ := m["prompt"].(string)
		return fmt.Sprintf("⚠️ Permission: %s\n   %s\n", toolName, prompt)
	case "result":
		return "\n--- Session Complete ---\n"
	default:
		return ""
	}
}

func extractContentBlocks(message any) string {
	text, _ := extractContentBlocksWithThinking(message)
	return text
}

func extractContentBlocksWithThinking(message any) (string, string) {
	switch v := message.(type) {
	case string:
		return v, ""
	case map[string]any:
		content, ok := v["content"]
		if !ok {
			return "", ""
		}
		contentArr, ok := content.([]any)
		if !ok {
			return "", ""
		}
		var texts, thinkings []string
		for _, block := range contentArr {
			blockMap, _ := block.(map[string]any)
			if blockMap == nil {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			t, _ := blockMap["text"].(string)
			th, _ := blockMap["thinking"].(string)
			if blockType == "text" && t != "" {
				texts = append(texts, t)
			}
			if blockType == "thinking" && th != "" {
				thinkings = append(thinkings, th)
			}
		}
		return strings.Join(texts, ""), strings.Join(thinkings, "")
	}
	return "", ""
}
