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
	logWindow *LogWindow // 可视化日志窗口（Windows）
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
	// 如果需要可视化终端，创建日志窗口
	if r.req.VisibleTerminal {
		logWin, err := NewLogWindow(r.sessionID)
		if err != nil {
			// 日志窗口创建失败不影响核心功能，只记录错误
			r.errors <- fmt.Errorf("create log window failed (non-fatal): %w", err)
		} else {
			r.logWindow = logWin
			fmt.Fprintf(r.logWindow, "[INFO] Starting Claude CLI...\n")
			fmt.Fprintf(r.logWindow, "[INFO] Command: %s %v\n\n", r.req.Command, r.req.Args)
		}
	}

	// 从 settings 文件读取环境变量（--settings 在 --print 模式下不工作）
	settingsEnv := extractSettingsEnv(r.req.Args)

	// Windows 上需要使用 cmd /c 来启动 npm 安装的命令
	command := r.req.Command
	var args []string
	// 过滤掉 --settings 参数，改用环境变量
	filteredArgs := filterSettingsArgs(r.req.Args)
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(command), ".exe") {
		args = []string{"/c", command, "--print", "--verbose", "--output-format", "stream-json", "--input-format", "stream-json", "--permission-prompt-tool", "stdio"}
		args = append(args, filteredArgs...)
		command = "cmd"
	} else {
		args = append([]string{"--print", "--verbose", "--output-format", "stream-json", "--input-format", "stream-json", "--permission-prompt-tool", "stdio"}, filteredArgs...)
	}

	cmd := exec.CommandContext(r.ctx, command, args...)
	if r.req.CWD != "" {
		cmd.Dir = r.req.CWD
	}
	// 合并环境变量：系统环境 + settings 文件中的环境变量
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

	// 记录用户输入到日志窗口
	if r.logWindow != nil && len(p) > 0 {
		content := strings.TrimRight(string(p), "\r\n")
		fmt.Fprintf(r.logWindow, "[USER INPUT] %s\n", content)
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

// extractSettingsEnv 从 --settings 参数中读取 settings 文件并提取环境变量。
// settings 文件格式：{ "env": { "KEY": "value", ... }, ... }
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

	// 展开环境变量（如 $HOME）
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

	// 展开环境变量中的引用
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

// settingsFilePath 从参数中提取 settings 文件路径。
func settingsFilePath(args []string) string {
	for i, arg := range args {
		if arg == "--settings" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
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

		// 输出格式化内容到日志窗口
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
			select {
			case r.errors <- errors.New("events channel full, dropping chunk"):
			default:
			}
		}
	}
}

// formatStreamJSONForLog 将 Claude stream-json 的单行解析为人类可读的文本。
func formatStreamJSONForLog(line []byte) string {
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return fmt.Sprintf("[RAW] %s\n", string(line))
	}

	typ, _ := m["type"].(string)
	switch typ {
	case "system":
		subtype, _ := m["subtype"].(string)
		if subtype == "hook_started" {
			return "" // 跳过 hook 启动事件
		}
		if subtype == "hook_response" {
			return "" // 跳过 hook 响应事件
		}
		return fmt.Sprintf("[SYSTEM] %s\n", subtype)
	case "assistant_message":
		msg := m["message"]
		text := extractAssistantText(msg)
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
		return "" // 忽略未知类型
	}
}

// extractAssistantText 从 assistant_message 的 message 字段提取文本。
func extractAssistantText(message any) string {
	switch v := message.(type) {
	case string:
		return v
	case map[string]any:
		content, ok := v["content"]
		if !ok {
			return ""
		}
		contentArr, ok := content.([]any)
		if !ok {
			return ""
		}
		var parts []string
		for _, block := range contentArr {
			blockMap, ok := block.(map[string]any)
			if !ok {
				continue
			}
			blockType, _ := blockMap["type"].(string)
			text, _ := blockMap["text"].(string)
			if blockType == "text" && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	}
	return ""
}

func (r *ClaudeRunner) readStderr(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 同时输出到日志窗口
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
	logWin := r.logWindow
	r.mu.Unlock()

	// 关闭日志窗口
	if logWin != nil {
		logWin.Close()
	}

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
