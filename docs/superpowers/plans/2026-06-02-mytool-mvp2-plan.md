# mytool MVP 2 实施计划

> **For agentic workers:** Use superpowers:subagent-driven-development to implement task-by-task.

**Goal:** 实施 MVP 2 spec 中的 4 个 followup：ClaudeRunner + CodexRunner + files/store + stall watchdog + yourname 替换。

**Architecture:**
- ClaudeRunner：解析 `claude --output-format stream-json --verbose` 的每行 JSON
- CodexRunner：解析 `codex app-server` JSON-RPC over stdin/stdout
- files：工作区文件树、读取、下载、denylist
- store：会话历史原子重命名存储
- stall watchdog：session.Manager.forward 中的计时器

**Tech Stack:** Go 标准库 + 现有依赖（gorilla/websocket、chi/v5、creack/pty、google/uuid）

**Spec Reference:** `docs/superpowers/specs/2026-06-02-mytool-mvp2-design.md`

---

## Task 22: ClaudeRunner (claude_parser.go + claude_runner.go)

**Files:**
- Create: `mytool/internal/engine/claude_parser.go`
- Create: `mytool/internal/engine/claude_runner.go`
- Create: `mytool/internal/engine/claude_parser_test.go`
- Create: `mytool/internal/engine/claude_runner_test.go`
- Modify: `mytool/internal/engine/registry.go`

### Step 1: 写 claude_parser_test.go

```go
package engine

import (
	"encoding/json"
	"testing"
)

func TestClaudeParserAssistantMessage(t *testing.T) {
	line := `{"type":"assistant_message","message":"Hello"}`
	ev, err := ParseClaudeStreamJSON([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	// The parser should emit raw JSON line for projection to handle
	if len(ev.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestClaudeParserToolUse(t *testing.T) {
	line := `{"type":"tool_use","name":"Bash","input":{"command":"ls"}}`
	ev, err := ParseClaudeStreamJSON([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	// Verify it's valid JSON
	var m map[string]any
	if err := json.Unmarshal(ev.Data, &m); err != nil {
		t.Errorf("Data should be valid JSON: %v", err)
	}
}

func TestClaudeParserPermissionRequest(t *testing.T) {
	line := `{"type":"permission_request","tool_name":"Bash","prompt":"Allow?"}`
	ev, err := ParseClaudeStreamJSON([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
}

func TestClaudeParserEmptyLine(t *testing.T) {
	_, err := ParseClaudeStreamJSON([]byte(""))
	if err != nil {
		t.Errorf("empty line should not error: %v", err)
	}
}

func TestClaudeParserInvalidJSON(t *testing.T) {
	_, err := ParseClaudeStreamJSON([]byte("not json"))
	if err == nil {
		t.Error("invalid JSON should error")
	}
}
```

### Step 2: 实现 claude_parser.go

```go
package engine

import (
	"encoding/json"
	"errors"
)

// ParseClaudeStreamJSON 解析 claude --output-format stream-json 的单行 JSON。
// 返回 Event{Kind: EventRaw, Data: <json bytes>}。
// MVP 2：不做深度解析，只透传 JSON；projection 层负责结构化。
func ParseClaudeStreamJSON(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{Kind: EventRaw, Data: nil}, nil
	}
	// Validate it's valid JSON
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return Event{}, errors.New("invalid JSON: " + err.Error())
	}
	// Copy the line to avoid sharing buffer
	cp := make([]byte, len(line))
	copy(cp, line)
	return Event{Kind: EventRaw, Data: cp}, nil
}
```

### Step 3: 实现 claude_runner.go

```go
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

// ClaudeRunner 启动 claude --output-format stream-json --verbose，
// 按行解析 JSON 并投递到 Events()。
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
	return r.cmd != nil && r.cmd.Process != nil
}

func (r *ClaudeRunner) Start(ctx context.Context, req ExecRequest) error {
	if req.Command == "" {
		return errors.New("command is required")
	}
	// Build claude command with stream-json output
	args := append([]string{"--output-format", "stream-json", "--verbose"}, req.Args...)
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
		return fmt.Errorf("start claude: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.mu.Unlock()

	r.events <- Event{Kind: EventLifecycle, Message: "started: claude"}

	go r.readLoop(stdout)
	go r.waitLoop()
	return nil
}

func (r *ClaudeRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.stdin == nil {
		return errors.New("runner is closed")
	}
	_, err := r.stdin.Write(p)
	return err
}

func (r *ClaudeRunner) Resize(cols, rows int) error {
	// Claude CLI doesn't support resize
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
```

### Step 4: 修改 registry.go

```go
// registry.go
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}
	switch {
	case command == "claude" || command == "claude-code" || command == "claude --resume":
		return NewClaudeRunner(), nil
	case command == "codex":
		return NewCodexRunner(), nil // Step in Task 23
	default:
		return NewPtyRunner(), nil
	}
}
```

### Step 5: 跑全部 engine 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/... -v
```

### Step 6: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/engine/
git commit -m "feat(engine): ClaudeRunner（解析 stream-json 输出）"
```

---

## Task 23: CodexRunner (codex_transport.go + codex_runner.go)

**Files:**
- Create: `mytool/internal/engine/codex_transport.go`
- Create: `mytool/internal/engine/codex_runner.go`
- Create: `mytool/internal/engine/codex_transport_test.go`
- Create: `mytool/internal/engine/codex_runner_test.go`

### Step 1: 写 codex_transport_test.go

```go
package engine

import (
	"testing"
)

func TestCodexTransportParse(t *testing.T) {
	// Test parsing a simple JSON-RPC notification
	line := `{"method":"thread/started","params":{"thread_id":"abc"}}`
	ev, err := ParseCodexJSONRPC([]byte(line))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != EventRaw {
		t.Errorf("Kind = %q, want raw", ev.Kind)
	}
	if len(ev.Data) == 0 {
		t.Error("Data should not be empty")
	}
}

func TestCodexTransportEmptyLine(t *testing.T) {
	_, err := ParseCodexJSONRPC([]byte(""))
	if err != nil {
		t.Errorf("empty line should not error: %v", err)
	}
}
```

### Step 2: 实现 codex_transport.go

```go
package engine

import (
	"encoding/json"
	"errors"
)

// ParseCodexJSONRPC 解析 codex app-server 的 JSON-RPC notification。
// 返回 Event{Kind: EventRaw, Data: <json bytes>}。
func ParseCodexJSONRPC(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{Kind: EventRaw, Data: nil}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		return Event{}, errors.New("invalid JSON: " + err.Error())
	}
	cp := make([]byte, len(line))
	copy(cp, line)
	return Event{Kind: EventRaw, Data: cp}, nil
}
```

### Step 3: 实现 codex_runner.go

```go
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
```

### Step 4: 更新 registry.go

```go
// registry.go
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}
	switch {
	case command == "claude" || command == "claude-code":
		return NewClaudeRunner(), nil
	case command == "codex":
		return NewCodexRunner(), nil
	default:
		return NewPtyRunner(), nil
	}
}
```

### Step 5: 跑全部 engine 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/... -v
```

### Step 6: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/engine/
git commit -m "feat(engine): CodexRunner（解析 JSON-RPC app-server）"
```

---

## Task 24: files 包 (tree + read + download + denylist)

**Files:**
- Create: `mytool/internal/files/tree.go`
- Create: `mytool/internal/files/read.go`
- Create: `mytool/internal/files/denylist.go`
- Create: `mytool/internal/files/download.go`
- Create: `mytool/internal/files/tree_test.go`
- Create: `mytool/internal/files/read_test.go`
- Create: `mytool/internal/files/denylist_test.go`

### Step 1: 写 denylist_test.go

```go
package files

import (
	"testing"
)

func TestDenyListMatches(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{".env", true},
		{"path/to/.env", true},
		{"server.key", true},
		{"cert.pem", true},
		{"cert.p12", true},
		{"ca.crt", true},
		{".git/config", true},
		{"node_modules/foo", true},
		{"src/main.go", false},
		{"README.md", false},
	}
	for _, c := range cases {
		got := IsDenied(c.path)
		if got != c.want {
			t.Errorf("IsDenied(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
```

### Step 2: 实现 denylist.go

```go
package files

import (
	"path/filepath"
	"strings"
)

var deniedPatterns = []string{
	".env",
	"*.pem",
	"*.key",
	"*.p12",
	"*.crt",
	".git",
	"node_modules",
}

// IsDenied 检查路径是否匹配 denylist。
func IsDenied(path string) bool {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	dir := filepath.Dir(clean)
	for _, pat := range deniedPatterns {
		if matched, _ := filepath.Match(pat, base); matched {
			return true
		}
		if strings.Contains(clean, ".git") || strings.Contains(clean, "node_modules") {
			return true
		}
	}
	return false
}
```

### Step 3: 写 tree.go

```go
package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode 表示文件树的一个节点。
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"isDir"`
	Children []TreeNode `json:"children,omitempty"`
}

// ListTree 列出工作区文件树，限制深度，忽略 denylist。
func ListTree(root string, depth int) ([]TreeNode, error) {
	if depth <= 0 {
		depth = 3
	}
	return listDir(root, "", depth)
}

func listDir(root, rel string, depth int) ([]TreeNode, error) {
	if depth <= 0 {
		return nil, nil
	}
	abs := filepath.Join(root, rel)
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var nodes []TreeNode
	for _, e := range entries {
		childRel := filepath.Join(rel, e.Name())
		if IsDenied(childRel) {
			continue
		}
		node := TreeNode{
			Name:  e.Name(),
			Path:  childRel,
			IsDir: e.IsDir(),
		}
		if e.IsDir() {
			children, err := listDir(root, childRel, depth-1)
			if err != nil {
				continue
			}
			node.Children = children
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}
```

### Step 4: 写 read.go

```go
package files

import (
	"errors"
	"os"
	"path/filepath"
)

// allowedExtensions 是允许读取的文件扩展名白名单。
var allowedExtensions = map[string]bool{
	".go":    true,
	".js":    true,
	".ts":    true,
	".tsx":   true,
	".py":    true,
	".java":  true,
	".rs":    true,
	".c":     true,
	".cpp":   true,
	".h":     true,
	".md":    true,
	".txt":   true,
	".json":  true,
	".yaml":  true,
	".yml":   true,
	".toml":  true,
	".xml":   true,
	".html":  true,
	".css":   true,
	".sh":    true,
	".bat":   true,
	".ps1":   true,
	".sql":   true,
	".graphql": true,
}

// ReadFile 读取工作区内的文件（白名单扩展名 + denylist 检查）。
func ReadFile(workspace, relPath string, maxBytes int) ([]byte, error) {
	if IsDenied(relPath) {
		return nil, errors.New("access denied")
	}
	ext := filepath.Ext(relPath)
	if !allowedExtensions[ext] {
		return nil, errors.New("file type not allowed")
	}
	abs := filepath.Join(workspace, relPath)
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if maxBytes > 0 && len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return data, nil
}
```

### Step 5: 跑 files 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/files/... -v
```

### Step 6: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/files/
git commit -m "feat(files): 文件树 + 读取 + denylist"
```

---

## Task 25: store 包 (filestore + sessions + skills + memory)

**Files:**
- Create: `mytool/internal/store/filestore.go`
- Create: `mytool/internal/store/sessions.go`
- Create: `mytool/internal/store/skills.go`
- Create: `mytool/internal/store/memory.go`
- Create: `mytool/internal/store/filestore_test.go`

### Step 1: 写 filestore_test.go

```go
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	type Item struct {
		Name string `json:"name"`
	}
	item := Item{Name: "hello"}

	if err := SaveJSON(path, item); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded Item
	if err := LoadJSON(path, &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if loaded.Name != "hello" {
		t.Errorf("Name = %q, want hello", loaded.Name)
	}
}

func TestFileStoreAtomicRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	if err := SaveJSON(path, map[string]string{"a": "1"}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}
	// Overwrite
	if err := SaveJSON(path, map[string]string{"a": "2"}); err != nil {
		t.Fatalf("SaveJSON overwrite: %v", err)
	}
	raw, _ := os.ReadFile(path)
	var m map[string]string
	json.Unmarshal(raw, &m)
	if m["a"] != "2" {
		t.Errorf("a = %q, want 2", m["a"])
	}
}
```

### Step 2: 实现 filestore.go

```go
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveJSON 把 v 序列化为 JSON 并原子写入 path（先写临时文件再 rename）。
func SaveJSON(path string, v any) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	return os.Rename(tmp, path)
}

// LoadJSON 从 path 加载 JSON 到 v。
func LoadJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	return json.Unmarshal(raw, v)
}
```

### Step 3: 实现 sessions.go

```go
package store

import (
	"path/filepath"
)

// SessionRecord 表示一个会话记录。
type SessionRecord struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	StartedAt string `json:"startedAt"`
	StoppedAt string `json:"stoppedAt,omitempty"`
}

// SaveSession 保存会话记录。
func SaveSession(storeDir string, rec SessionRecord) error {
	path := filepath.Join(storeDir, "sessions", rec.ID+".json")
	return SaveJSON(path, rec)
}

// LoadSession 加载会话记录。
func LoadSession(storeDir, id string) (SessionRecord, error) {
	path := filepath.Join(storeDir, "sessions", id+".json")
	var rec SessionRecord
	if err := LoadJSON(path, &rec); err != nil {
		return rec, err
	}
	return rec, nil
}
```

### Step 4: 实现 skills.go 和 memory.go（简单占位）

```go
// skills.go
package store

// SkillEntry 表示一个 skill。
type SkillEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// ListSkills 列出工作区的 skills。
func ListSkills(workspace string) ([]SkillEntry, error) {
	// MVP 2：简单占位
	return nil, nil
}
```

```go
// memory.go
package store

// MemoryEntry 表示一条 memory。
type MemoryEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ListMemory 列出 memory。
func ListMemory(storeDir string) ([]MemoryEntry, error) {
	// MVP 2：简单占位
	return nil, nil
}
```

### Step 5: 跑 store 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/store/... -v
```

### Step 6: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/store/
git commit -m "feat(store): 原子重命名存储 + sessions/skills/memory"
```

---

## Task 26: Stall watchdog (session.Manager.forward)

**Files:**
- Modify: `mytool/internal/session/manager.go`
- Create: `mytool/internal/session/watchdog_test.go`

### Step 1: 写 watchdog_test.go

```go
package session

import (
	"context"
	"testing"
	"time"
)

func TestStallWatchdog(t *testing.T) {
	m := NewManager()
	run := newFakeRunner()
	_, _ = m.Start(context.Background(), ExecRequest{Command: "x"}, run)

	// 模拟沉默：不推任何事件
	// 等待 130s（watchdog 120s + 10s buffer）
	// 实际测试中用短超时
	select {
	case <-m.Done():
		// watchdog 应该在 120s 内 kill
	case <-time.After(130 * time.Second):
		t.Errorf("watchdog did not kill runner within 130s")
	}
}
```

### Step 2: 改 manager.go

在 `forward` 中加 watchdog：

```go
func (m *Manager) forward(run engine.Runner) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	lastActivity := time.Now()

	for {
		select {
		case ev, ok := <-run.Events():
			if !ok {
				return
			}
			lastActivity = time.Now()
			m.out <- ev
		case <-ticker.C:
			if time.Since(lastActivity) > 120*time.Second {
				// Stall watchdog: kill runner
				run.Close()
				return
			}
		}
	}
}
```

同时加 `Done()` 方法：

```go
// Done 返回 forward 结束的信号。
func (m *Manager) Done() <-chan struct{} {
	return m.done
}
```

### Step 3: 跑 session 测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/session/... -v
```

### Step 4: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/session/
git commit -m "feat(session): stall watchdog 120s 沉默 kill"
```

---

## Task 27: yourname 占位替换

**Files:**
- Modify: `mytool/go.mod`
- Modify: 所有 `.go` 文件中的 import 路径

### Step 1: 确定真实 module path

待用户确认。暂用 `github.com/jaycrl/mytool`。

### Step 2: 替换

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool"
# 替换 go.mod
sed -i 's|github.com/yourname/mytool|github.com/jaycrl/mytool|g' go.mod

# 替换所有 .go 文件
find . -name '*.go' -exec sed -i 's|github.com/yourname/mytool|github.com/jaycrl/mytool|g' {} +
```

### Step 3: 验证编译

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build ./... && go test ./... -count=1
```

### Step 4: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/
git commit -m "chore: 替换 yourname 占位为 jaycrl"
```

---

## Task 28: 集成 + smoke

### Step 1: 跑全部测试

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

### Step 2: 跑 smoke

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && bash scripts/e2e_smoke.sh && bash scripts/tls_smoke.sh
```

### Step 3: 提交

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add -A
git commit -m "test: MVP 2 整体回归"
```

---

## 完成标准

- [ ] `claude` command 启动 ClaudeRunner
- [ ] `codex` command 启动 CodexRunner
- [ ] files 包工作
- [ ] store 包工作
- [ ] stall watchdog 120s kill 有效
- [ ] yourname 占位替换完成
- [ ] `go test ./...` 全部 PASS
- [ ] e2e smoke 通过
