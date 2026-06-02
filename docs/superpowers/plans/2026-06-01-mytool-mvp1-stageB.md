# mytool MVP 1 阶段 B：engine + projection + session

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 AI 引擎抽象（Runner interface）、PTY 通用实现、引擎注册表、最小投影层、活跃 session 管理。让一个"PTY 跑 echo hello"的会话能被 structured event 流式读出。

**Architecture:** 阶段 B 引入 `internal/{engine,projection,session}` 三个包。engine 抽象 `Runner` interface；projection 订阅 runner 的原始行 → 转 `projection.Event`；session 持有单个 runner + 转发事件到 hub。MVP 1 阶段只覆盖 PTY；ClaudeRunner/CodexRunner 留到 MVP 2/3。

**Tech Stack:**
- `github.com/creack/pty` v1.1.24（PTY 启动）
- `github.com/google/uuid` v1.6.0（session id）
- 标准库 `testing` + 临时脚本做集成测试

**Spec Reference:** spec §6 (AI 引擎抽象) + §7.1 (投影目标) + §6.4 (stall watchdog 设计借鉴) + §2 (engine/projection/session 包结构)

**前置：** 阶段 A 已完成（`internal/{logx,config,auth}` 通过测试）。

---

## Task 5: internal/engine（Runner interface + PTY 实现）

**Files:**
- Modify: `mytool/go.mod`（新增 `creack/pty` 依赖）
- Create: `mytool/internal/engine/runner.go`
- Create: `mytool/internal/engine/pty_runner.go`
- Test: `mytool/internal/engine/pty_runner_test.go`

- [ ] **Step 1: 添加 creack/pty 依赖**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go get github.com/creack/pty@v1.1.24
```

预期：`go.mod` 与 `go.sum` 出现新行，无错误。

- [ ] **Step 2: 写 pty_runner_test.go**

文件：`mytool/internal/engine/pty_runner_test.go`

```go
package engine

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestPtyRunnerEcho(t *testing.T) {
	r := NewPtyRunner()
	req := ExecRequest{
		Command: "printf",
		Args:    []string{"hello-pty\\n"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := r.Start(ctx, req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer r.Close()

	// 等到 stdout 出现目标字符串
	deadline := time.After(3 * time.Second)
	var got string
	for {
		select {
		case ev, ok := <-r.Events():
			if !ok {
				t.Fatalf("events channel closed before getting expected output")
			}
			if ev.Kind == EventRaw {
				got += string(ev.Data)
				if strings.Contains(got, "hello-pty") {
					return // success
				}
			}
		case err := <-r.Errors():
			t.Fatalf("unexpected error: %v", err)
		case <-deadline:
			t.Fatalf("timeout waiting for output, got so far: %q", got)
		}
	}
}

func TestPtyRunnerStartRejectsEmptyCommand(t *testing.T) {
	r := NewPtyRunner()
	if err := r.Start(context.Background(), ExecRequest{Command: ""}); err == nil {
		t.Errorf("Start with empty command should fail")
	}
}

func TestPtyRunnerLifecycle(t *testing.T) {
	r := NewPtyRunner()
	req := ExecRequest{Command: "sleep", Args: []string{"0.3"}}
	if err := r.Start(context.Background(), req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// 给 50ms 启动时间
	time.Sleep(50 * time.Millisecond)
	select {
	case <-r.Done():
		t.Errorf("Done() should not be ready yet for a 0.3s sleep")
	default:
	}
	// 等到完成
	select {
	case <-r.Done():
	case <-time.After(2 * time.Second):
		t.Errorf("Done() should fire after sleep ends")
	}
	_ = r.Close()
}
```

- [ ] **Step 3: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/...
```

预期：FAIL（包内还没有任何文件）。

- [ ] **Step 4: 实现 runner.go**

文件：`mytool/internal/engine/runner.go`

```go
// Package engine 定义 AI 引擎抽象：Runner interface + 通用 PTY 实现 + 注册表。
// 不同 AI CLI（claude / codex / 任意 LLM CLI）通过不同 Runner 实现接入。
package engine

import "context"

// EventKind 区分事件的来源/类型。
type EventKind string

const (
	// EventRaw 表示原始 PTY 输出/JSON-RPC 字节流片段。
	EventRaw EventKind = "raw"
	// EventLifecycle 表示 runner 启动/退出/错误等生命周期事件。
	EventLifecycle EventKind = "lifecycle"
)

// Event 是 Runner 通过 Events() 通道发出的最小信息单元。
type Event struct {
	Kind    EventKind
	Data    []byte // 仅 EventRaw 携带原始字节
	Message string // EventLifecycle 的描述
}

// ExecRequest 启动一个 runner 所需的最小参数。
type ExecRequest struct {
	Command string
	Args    []string
	CWD     string
	Env     []string
	Cols    int
	Rows    int
}

// InteractiveStateProvider 报告 runner 是否准备好接收用户输入。
type InteractiveStateProvider interface {
	CanAcceptInteractiveInput() bool
}

// TurnStateProvider 报告 runner 是否正在一个 assistant turn 中。
type TurnStateProvider interface {
	HasActiveTurn() bool
}

// Runner 是所有 AI 引擎实现的统一接口。
// MVP 1 阶段只要求 PTY 实现；ClaudeRunner/CodexRunner 留到 MVP 2/3。
type Runner interface {
	Start(ctx context.Context, req ExecRequest) error
	Write(p []byte) error
	Resize(cols, rows int) error
	Close() error

	Events() <-chan Event
	Errors() <-chan error
	Done() <-chan struct{}

	SessionID() string

	InteractiveStateProvider
	TurnStateProvider
}
```

- [ ] **Step 5: 实现 pty_runner.go**

文件：`mytool/internal/engine/pty_runner.go`

```go
package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/google/uuid"
)

// PtyRunner 是基于 PTY 的通用 Runner。任何能把 stdin/stdout 当成 TTY 的
// 程序都能跑（MVP 1 阶段没有 stream-json 解析，只透传原始字节）。
type PtyRunner struct {
	cmd        *exec.Cmd
	ptyFile    *os.File
	events     chan Event
	errors     chan error
	done       chan struct{}
	sessionID  string
	mu         sync.Mutex
	closed     bool
	hasTurn    bool   // 简化版：进程存在即视为 active turn
	canInput   bool   // PTY 模式默认可输入
}

// NewPtyRunner 构造一个未启动的 PtyRunner。
func NewPtyRunner() *PtyRunner {
	return &PtyRunner{
		events:    make(chan Event, 64),
		errors:    make(chan error, 8),
		done:      make(chan struct{}),
		canInput:  true,
		sessionID: "pty_" + uuid.NewString(),
	}
}

// SessionID 返回 runner 的 session id。
func (r *PtyRunner) SessionID() string { return r.sessionID }

// Events 返回原始事件通道。
func (r *PtyRunner) Events() <-chan Event { return r.events }

// Errors 返回错误通道。
func (r *PtyRunner) Errors() <-chan error { return r.errors }

// Done 在进程退出时关闭。
func (r *PtyRunner) Done() <-chan struct{} { return r.done }

// Write 把 p 写入 PTY stdin。
func (r *PtyRunner) Write(p []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.ptyFile == nil {
		return errors.New("runner is closed")
	}
	_, err := r.ptyFile.Write(p)
	return err
}

// Resize 调整 PTY 窗口大小。
func (r *PtyRunner) Resize(cols, rows int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ptyFile == nil {
		return errors.New("pty not started")
	}
	return pty.Setsize(r.ptyFile, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

// CanAcceptInteractiveInput PTY runner 总是可输入。
func (r *PtyRunner) CanAcceptInteractiveInput() bool { return r.canInput }

// HasActiveTurn 进程在跑即视为有 turn。
func (r *PtyRunner) HasActiveTurn() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil && r.cmd.Process != nil
}

// Start 启动 PTY 进程并启动读循环。
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

	// 推一个 lifecycle 事件
	r.events <- Event{Kind: EventLifecycle, Message: "started: " + req.Command}

	// 启动读循环
	go r.readLoop(ptmx)
	// 启动等待循环
	go r.waitLoop()
	return nil
}

// readLoop 从 PTY 读字节并投递到 events。
func (r *PtyRunner) readLoop(f *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			// 复制一份再投递，避免下游读到共享底层数组
			cp := make([]byte, n)
			copy(cp, buf[:n])
			select {
			case r.events <- Event{Kind: EventRaw, Data: cp}:
			default:
				// 通道满：丢下一条错以提示背压
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

// waitLoop 等待进程结束。
func (r *PtyRunner) waitLoop() {
	err := r.cmd.Wait()
	r.mu.Lock()
	r.hasTurn = false
	r.mu.Unlock()
	if err != nil {
		r.errors <- err
		r.events <- Event{Kind: EventLifecycle, Message: "exited: " + err.Error()}
	} else {
		r.events <- Event{Kind: EventLifecycle, Message: "exited"}
	}
	close(r.done)
}

// Close 关闭 PTY 与进程。
func (r *PtyRunner) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.ptyFile.Close()
	cmd := r.cmd
	r.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
```

- [ ] **Step 6: 跑 engine 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/... -v
```

预期：3 个测试 PASS（`printf` 在 Linux/macOS/Windows 行为不同，Windows 上 `printf` 由 Git Bash 或 PowerShell 提供；如本地非 bash 环境，可改用 `cmd /c echo` 调整；CI 跑 Linux 时本测试稳定通过）。

如果 Windows 平台 `printf` 不可用，可临时把 `TestPtyRunnerEcho` 改为：

```go
req := ExecRequest{Command: "cmd", Args: []string{"/c", "echo", "hello-pty"}}
```

并把 expected 字符串从 `hello-pty\n` 改为 `hello-pty`（Windows `echo` 自带换行）。

- [ ] **Step 7: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/go.mod mytool/go.sum mytool/internal/engine/
git commit -m "feat(engine): Runner interface + PtyRunner 通用实现"
```

---

## Task 6: internal/engine/registry（按 command 选实现）

**Files:**
- Create: `mytool/internal/engine/registry.go`
- Test: `mytool/internal/engine/registry_test.go`

- [ ] **Step 1: 写 registry_test.go**

文件：`mytool/internal/engine/registry_test.go`

```go
package engine

import "testing"

func TestRegistryReturnsPtyForUnknown(t *testing.T) {
	r, err := NewRunner("aichat", ExecRequest{})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if _, ok := r.(*PtyRunner); !ok {
		t.Errorf("unknown command should fall back to PtyRunner, got %T", r)
	}
	_ = r.Close()
}

func TestRegistryRejectsEmpty(t *testing.T) {
	if _, err := NewRunner("", ExecRequest{}); err == nil {
		t.Errorf("NewRunner(\"\") should fail")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/... -run TestRegistry
```

预期：FAIL（`undefined: NewRunner`）。

- [ ] **Step 3: 实现 registry.go**

文件：`mytool/internal/engine/registry.go`

```go
package engine

import "errors"

// NewRunner 根据 command 返回合适的 Runner 实现。
// MVP 1 阶段：所有非空 command 都返回 PtyRunner。
// 后续 MVP：把 "claude" 路由到 ClaudeRunner、"codex" 路由到 CodexRunner。
func NewRunner(command string, _ ExecRequest) (Runner, error) {
	if command == "" {
		return nil, errors.New("engine: command is required")
	}
	// MVP 2+ 将按 command 选实现。这里所有走 PTY。
	return NewPtyRunner(), nil
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/engine/... -v -run TestRegistry
```

预期：2 个测试 PASS。

- [ ] **Step 5: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/engine/registry.go mytool/internal/engine/registry_test.go
git commit -m "feat(engine): registry 默认所有 command 走 PTY"
```

---

## Task 7: internal/projection（最小投影）

**Files:**
- Create: `mytool/internal/projection/event.go`
- Create: `mytool/internal/projection/raw.go`
- Test: `mytool/internal/projection/raw_test.go`

- [ ] **Step 1: 写 raw_test.go**

文件：`mytool/internal/projection/raw_test.go`

```go
package projection

import (
	"testing"
	"time"

	"mytool/internal/engine"
)

func TestRawTextEvent(t *testing.T) {
	in := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte("hello\n")},
		{Kind: engine.EventRaw, Data: []byte("world\n")},
		{Kind: engine.EventLifecycle, Message: "started"},
	}
	got := Project(in)
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	if got[0].Type != EventText || got[0].Text != "hello" {
		t.Errorf("got[0] = %+v, want text/hello", got[0])
	}
	if got[1].Type != EventText || got[1].Text != "world" {
		t.Errorf("got[1] = %+v, want text/world", got[1])
	}
	if got[2].Type != EventLifecycle || got[2].Message != "started" {
		t.Errorf("got[2] = %+v, want lifecycle/started", got[2])
	}
	if got[0].Time.IsZero() {
		t.Errorf("event should carry timestamp")
	}
	if got[0].SessionID == "" {
		t.Errorf("event should carry session id")
	}
}

func TestRawStripsCRLF(t *testing.T) {
	in := []engine.Event{
		{Kind: engine.EventRaw, Data: []byte("line\r\n")},
	}
	got := Project(in)
	if got[0].Text != "line" {
		t.Errorf("got[0].Text = %q, want %q", got[0].Text, "line")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/projection/...
```

预期：FAIL（包内无文件）。

- [ ] **Step 3: 实现 event.go**

文件：`mytool/internal/projection/event.go`

```go
// Package projection 把 engine.Runner 输出的原始字节流翻译成结构化事件。
// MVP 1 阶段：只做 raw 文本事件与 lifecycle 透传。
// 后续 MVP：加入 diff / permission / plan / context_window 等高级投影。
package projection

import "time"

// EventType 区分投影后的事件类型。
type EventType string

const (
	EventText      EventType = "text"
	EventLifecycle EventType = "lifecycle"
	// 后续扩展：EventDiff / EventPermission / EventPlan / EventContextWindow / EventError
)

// Event 是投影后的事件（前端订阅的契约）。
type Event struct {
	Type      EventType
	SessionID string
	Time      time.Time
	Text      string    // EventText
	Message   string    // EventLifecycle
}
```

- [ ] **Step 4: 实现 raw.go**

文件：`mytool/internal/projection/raw.go`

```go
package projection

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"mytool/internal/engine"
)

// Project 把 engine.Event 序列翻译为投影事件。sessionID 自动注入（若原始事件有）。
// MVP 1：仅做 raw 文本与 lifecycle 透传。
func Project(in []engine.Event) []Event {
	out := make([]Event, 0, len(in))
	sid := "sess_" + uuid.NewString()
	now := time.Now().UTC()
	for _, ev := range in {
		switch ev.Kind {
		case engine.EventRaw:
			text := strings.TrimRight(string(ev.Data), "\r\n")
			out = append(out, Event{
				Type:      EventText,
				SessionID: sid,
				Time:      now,
				Text:      text,
			})
		case engine.EventLifecycle:
			out = append(out, Event{
				Type:      EventLifecycle,
				SessionID: sid,
				Time:      now,
				Message:   ev.Message,
			})
		}
	}
	return out
}

// Stream 实时投影：从 input 通道读取 engine.Event 投到 output。
// MVP 1 阶段使用：session manager 包装 runner 事件后调用。
func Stream(input <-chan engine.Event, output chan<- Event) {
	sid := "sess_" + uuid.NewString()
	for ev := range input {
		switch ev.Kind {
		case engine.EventRaw:
			output <- Event{
				Type:      EventText,
				SessionID: sid,
				Time:      time.Now().UTC(),
				Text:      strings.TrimRight(string(ev.Data), "\r\n"),
			}
		case engine.EventLifecycle:
			output <- Event{
				Type:      EventLifecycle,
				SessionID: sid,
				Time:      time.Now().UTC(),
				Message:   ev.Message,
			}
		}
	}
	close(output)
}
```

- [ ] **Step 5: 跑测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/projection/... -v
```

预期：2 个测试 PASS。

- [ ] **Step 6: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/projection/
git commit -m "feat(projection): raw 文本 + lifecycle 透传（MVP 1 最小投影）"
```

---

## Task 8: internal/session（活跃 session 管理）

**Files:**
- Create: `mytool/internal/session/manager.go`
- Test: `mytool/internal/session/manager_test.go`

- [ ] **Step 1: 写 manager_test.go**

文件：`mytool/internal/session/manager_test.go`

```go
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
func (f *fakeRunner) Write(p []byte) error                 { return nil }
func (f *fakeRunner) Resize(c, r int) error                { return nil }
func (f *fakeRunner) Close() error                         { f.mu.Lock(); defer f.mu.Unlock(); f.closed = true; return nil }
func (f *fakeRunner) Events() <-chan Event                 { return f.events }
func (f *fakeRunner) Errors() <-chan error                 { return f.errors }
func (f *fakeRunner) Done() <-chan struct{}                { return f.done }
func (f *fakeRunner) SessionID() string                    { return "fake" }
func (f *fakeRunner) CanAcceptInteractiveInput() bool      { return true }
func (f *fakeRunner) HasActiveTurn() bool                  { return true }

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
```

- [ ] **Step 2: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/session/...
```

预期：FAIL（包内无文件）。

- [ ] **Step 3: 实现 manager.go**

文件：`mytool/internal/session/manager.go`

```go
// Package session 管理单个活跃 runner 的生命周期与事件转发。
// MVP 1 阶段：仅一个活跃 session；后续可扩展 resume / permission router。
package session

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"mytool/internal/engine"
)

// Event 是 session 包暴露的最小事件类型（直接转 engine.Event）。
type Event = engine.Event

// ExecRequest 透传 engine.ExecRequest，避免上层 import engine。
type ExecRequest = engine.ExecRequest

// Manager 持有当前活跃 runner 与一条到订阅者的输出流。
type Manager struct {
	mu     sync.Mutex
	active engine.Runner
	sid    string
	out    chan Event
	err    chan error
	once   sync.Once
}

// NewManager 构造一个空 manager。
func NewManager() *Manager {
	return &Manager{
		out: make(chan Event, 64),
		err: make(chan error, 8),
	}
}

// Output 返回当前活跃 session 的事件流。Stop 后会被关闭。
func (m *Manager) Output() <-chan Event { return m.out }

// Errs 返回错误流。
func (m *Manager) Errs() <-chan error { return m.err }

// Start 启动新 session。要求当前无活跃 runner。
// 返回的 session id 用于跨引用。
func (m *Manager) Start(ctx context.Context, req ExecRequest, run engine.Runner) (string, error) {
	m.mu.Lock()
	if m.active != nil {
		m.mu.Unlock()
		return "", errors.New("session: another runner is already active")
	}
	m.active = run
	m.sid = "sess_" + uuid.NewString()
	m.mu.Unlock()

	if err := run.Start(ctx, req); err != nil {
		m.mu.Lock()
		m.active = nil
		m.mu.Unlock()
		return "", err
	}

	go m.forward(run)
	return m.sid, nil
}

// forward 把 runner 的事件复制到 manager 的 output。
func (m *Manager) forward(run engine.Runner) {
	for ev := range run.Events() {
		m.out <- ev
	}
	close(m.out)
	// 进程退出后清空 active
	m.mu.Lock()
	if m.active == run {
		m.active = nil
	}
	m.mu.Unlock()
}

// Stop 关闭当前活跃 session。
func (m *Manager) Stop() error {
	m.mu.Lock()
	run := m.active
	m.active = nil
	m.mu.Unlock()
	if run == nil {
		return nil
	}
	return run.Close()
}
```

- [ ] **Step 4: 跑测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/session/... -v
```

预期：3 个测试 PASS。

- [ ] **Step 5: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/internal/session/
git commit -m "feat(session): Manager 持有活跃 runner + 事件转发"
```

---

## 阶段 B 完成标准

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -v
```

预期：阶段 A 的 21 个 + 阶段 B 的 8 个 = 29 个测试全部 PASS。

git log 新增 4 个 commit：

```
feat(engine): Runner interface + PtyRunner 通用实现
feat(engine): registry 默认所有 command 走 PTY
feat(projection): raw 文本 + lifecycle 透传（MVP 1 最小投影）
feat(session): Manager 持有活跃 runner + 事件转发
```

阶段 B 完成后，仓库已具备：
- 跑任意 LLM CLI 的 PTY 通用 runner
- 引擎注册表（按 command 选实现）
- 原始行 → 文本事件的最简投影
- 单活跃 session 生命周期管理
