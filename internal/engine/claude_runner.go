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

// ClaudeRunner 启动 claude --print --output-format stream-json，
// 每次消息通过命令行参数传 prompt（不用 stdin 管道，更可靠）。
type ClaudeRunner struct {
	cmd       *exec.Cmd
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
	logWindow *LogWindow
	// 用于多轮对话的 session 恢复
	claudeSessionID string // --resume 参数使用的 Claude 内部 session id
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
	r.ctx = ctx
	r.req = req
	r.events <- Event{Kind: EventLifecycle, Message: "ready: claude (waiting for first message)"}
	return nil
}

// startProcess 启动 claude 子进程，用命令行参数传 prompt。
func (r *ClaudeRunner) startProcess(prompt string) error {
	settingsEnv := extractSettingsEnv(r.req.Args)
	filteredArgs := filterSettingsArgs(r.req.Args)

	// 构建基本参数：--print --verbose --output-format stream-json
	baseArgs := []string{"--print", "--verbose", "--output-format", "stream-json", "--permission-prompt-tool", "stdio"}

	// 如果是多轮对话，添加 --resume
	if r.claudeSessionID != "" {
		baseArgs = append(baseArgs, "--resume", r.claudeSessionID)
	}

	// 添加其他参数（排除 --settings）和 prompt
	baseArgs = append(baseArgs, filteredArgs...)
	if prompt != "" {
		baseArgs = append(baseArgs, prompt)
	}

	command := r.req.Command
	var args []string
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(command), ".exe") {
		args = []string{"/c", command}
		args = append(args, baseArgs...)
		command = "cmd"
	} else {
		args = baseArgs
	}

	r.events <- Event{Kind: EventLifecycle, Message: fmt.Sprintf("cmd: %s %s", command, strings.Join(args, " "))}

	cmd := exec.CommandContext(r.ctx, command, args...)
	if r.req.CWD != "" {
		cmd.Dir = r.req.CWD
	}
	// 设置环境变量
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

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.stdout = stdout
	r.started = true
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: claude"}

	go r.readLoop(stdout)
	go r.readStderr(stderr)
	go r.waitLoop()
	return nil
}

// Write 写入用户消息。
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

	// 记录用户输入
	if r.logWindow != nil {
		fmt.Fprintf(r.logWindow, "[USER INPUT] %s\n", content)
	}

	// 每次都启动新进程（或者第一次启动）
	r.mu.Lock()
	isRunning := r.started && r.cmd != nil && r.cmd.Process != nil
	r.mu.Unlock()

	if isRunning {
		// 如果已经在运行，先关闭旧进程
		r.killProcess()
		r.mu.Lock()
		r.started = false
		r.mu.Unlock()
	}

	return r.startProcess(content)
}

// killProcess 强制终止当前进程。
func (r *ClaudeRunner) killProcess() {
	r.mu.Lock()
	cmd := r.cmd
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}
}

func (r *ClaudeRunner) Resize(cols, rows int) error {
	return nil
}

func (r *ClaudeRunner) readLoop(stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// 检查是否包含 session_id（用于后续 --resume）
		r.captureSessionID(line)

		if r.logWindow != nil {
			fmt.Fprint(r.logWindow, formatStreamJSONForLog(line))
		}

		ev, err := ParseClaudeStreamJSON(line)
		if err != nil {
			r.errors <- fmt.Errorf("claude parse: %w", err)
			continue
		}
		select {
		case r.events <- ev:
		default:
		}
	}
}

// captureSessionID 从 result 事件中提取 session_id。
func (r *ClaudeRunner) captureSessionID(line []byte) {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return
	}
	if sid, ok := m["session_id"].(string); ok && sid != "" {
		r.claudeSessionID = sid
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
	err := r.cmd.Wait()
	r.mu.Lock()
	wasClosed := r.closed
	r.mu.Unlock()
	if err != nil && !wasClosed {
		r.errors <- err
		r.events <- Event{Kind: EventLifecycle, Message: "exited: " + err.Error()}
	} else {
		r.events <- Event{Kind: EventLifecycle, Message: "exited"}
	}
	// 注意：不关闭 channels，runner 可以重新启动
}

func (r *ClaudeRunner) Close() error {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	r.killProcess()
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
