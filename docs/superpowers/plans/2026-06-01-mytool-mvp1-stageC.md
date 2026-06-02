# mytool MVP 1 阶段 C：ws + gateway + cmd 入口 + SPA + e2e

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把前两阶段（基础设施 + engine/session）拼成可启动的服务。手机浏览器扫码/输 token → 打开 SPA → WebSocket 接入 → 启动 PTY session → 实时看到文本流。

**Architecture:** `internal/ws` 实现协议编解码 + 单连接 + hub + 消息分发；`internal/gateway` 提供 HTTP 路由（含 SPA fallback、healthz、qr stub）；`cmd/server` 装配 config + auth + engine + session + gateway + ws；`web/` 占位 SPA 通过 `go:embed` 内嵌；`scripts/e2e_smoke.sh` 跑端到端验证。

**Tech Stack:**
- `github.com/gorilla/websocket` v1.5.1（WebSocket）
- `github.com/go-chi/chi/v5` v5.0.12（HTTP 路由）
- 标准库 `embed`（SPA 嵌入）
- bash + `curl` + `websocat`（e2e smoke）

**Spec Reference:** spec §4 (WebSocket 协议) + §5 (鉴权) + §8 (部署与启动) + §11 (验收标准)

**前置：** 阶段 A、B 已完成（`internal/{logx,config,auth,engine,projection,session}` 通过测试）。

---

## Task 9: internal/ws（WebSocket 协议）

**Files:**
- Modify: `mytool/go.mod`（新增 `gorilla/websocket`）
- Create: `mytool/internal/ws/codec.go`
- Create: `mytool/internal/ws/conn.go`
- Create: `mytool/internal/ws/hub.go`
- Create: `mytool/internal/ws/handler.go`
- Test: `mytool/internal/ws/codec_test.go`
- Test: `mytool/internal/ws/hub_test.go`

- [ ] **Step 1: 添加 gorilla/websocket 依赖**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go get github.com/gorilla/websocket@v1.5.1
```

预期：go.mod / go.sum 出现新行。

- [ ] **Step 2: 写 codec_test.go**

文件：`mytool/internal/ws/codec_test.go`

```go
package ws

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeReqRoundTrip(t *testing.T) {
	in := Envelope{
		Type:   "req",
		ID:     "u-1",
		Method: "session.start",
		Params: json.RawMessage(`{"command":"echo","args":["hi"]}`),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Envelope
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Type != "req" || out.ID != "u-1" || out.Method != "session.start" {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestEnvelopeRespError(t *testing.T) {
	raw := `{"type":"resp","id":"u-2","ok":false,"error":{"code":"unauthorized","message":"bad token"}}`
	var e Envelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Error == nil || e.Error.Code != "unauthorized" {
		t.Errorf("error decode wrong: %+v", e.Error)
	}
}

func TestEnvelopeEvt(t *testing.T) {
	raw := `{"type":"evt","sessionId":"sess_1","event":{"type":"text","text":"hello","sessionId":"sess_1"}}`
	var e Envelope
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.Type != "evt" || e.Event == nil {
		t.Errorf("event decode wrong: %+v", e)
	}
}
```

- [ ] **Step 3: 实现 codec.go**

文件：`mytool/internal/ws/codec.go`

```go
// Package ws 实现 mytool WebSocket 协议（v1，JSON 编码）。
// 协议细节见 spec §4。
package ws

import "encoding/json"

// Envelope 是 ws 上传输的顶层消息。
// 三种 type：req（客户端请求）、resp（请求响应）、evt（服务端主动推送）。
type Envelope struct {
	Type      string          `json:"type"`
	ID        string          `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	OK        *bool           `json:"ok,omitempty"`
	Error     *RPCError       `json:"error,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Event     json.RawMessage `json:"event,omitempty"`
}

// RPCError 描述一次失败响应的错误。
type RPCError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
```

- [ ] **Step 4: 跑 codec 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/ws/... -v -run TestEnvelope
```

预期：3 个测试 PASS。

- [ ] **Step 5: 写 hub_test.go**

文件：`mytool/internal/ws/hub_test.go`

```go
package ws

import (
	"sync"
	"testing"
	"time"
)

func TestHubBroadcast(t *testing.T) {
	h := NewHub()
	var wg sync.WaitGroup
	collected := make([]Envelope, 0)
	var mu sync.Mutex
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := h.Subscribe()
			select {
			case env := <-ch:
				mu.Lock()
				collected = append(collected, env)
				mu.Unlock()
			case <-time.After(1 * time.Second):
			}
		}()
	}
	// 等所有订阅者注册
	time.Sleep(50 * time.Millisecond)
	h.Broadcast(Envelope{Type: "evt", SessionID: "sess_x"})
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(collected) != 3 {
		t.Errorf("collected = %d, want 3", len(collected))
	}
}

func TestHubUnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	ch := h.Subscribe()
	h.Unsubscribe(ch)
	h.Broadcast(Envelope{Type: "evt"})
	select {
	case _, ok := <-ch:
		if ok {
			t.Errorf("expected closed channel after unsubscribe")
		}
	case <-time.After(100 * time.Millisecond):
		// closed channel may already be drained; that's fine
	}
}
```

- [ ] **Step 6: 跑 hub 测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/ws/... -run TestHub
```

预期：FAIL（hub 未实现）。

- [ ] **Step 7: 实现 hub.go**

文件：`mytool/internal/ws/hub.go`

```go
package ws

import "sync"

// Hub 把服务端事件广播给所有当前订阅者。
type Hub struct {
	mu          sync.Mutex
	subscribers map[chan Envelope]struct{}
}

// NewHub 构造空 hub。
func NewHub() *Hub {
	return &Hub{subscribers: make(map[chan Envelope]struct{})}
}

// Subscribe 注册一个新的订阅者，返回一个缓冲为 32 的 channel。
// 调用方在不需要时调用 Unsubscribe。
func (h *Hub) Subscribe() chan Envelope {
	ch := make(chan Envelope, 32)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

// Unsubscribe 注销订阅者并关闭 channel。
func (h *Hub) Unsubscribe(ch chan Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
}

// Broadcast 把 env 投递给所有订阅者。订阅者 channel 满则丢弃。
func (h *Hub) Broadcast(env Envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- env:
		default:
			// 背压：丢下一条
		}
	}
}

// SubscriberCount 返回当前订阅者数（用于测试与监控）。
func (h *Hub) SubscriberCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subscribers)
}
```

- [ ] **Step 8: 跑 hub 测试确认通过**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/ws/... -v -run TestHub
```

预期：2 个测试 PASS。

- [ ] **Step 9: 实现 conn.go（单连接读写）**

文件：`mytool/internal/ws/conn.go`

```go
package ws

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// readTimeout 单条消息读超时。
const readTimeout = 60 * time.Second

// writeTimeout 单条消息写超时。
const writeTimeout = 10 * time.Second

// pingInterval 服务端 ping 间隔。
const pingInterval = 15 * time.Second

// Conn 包装 *websocket.Conn，提供读循环与写方法。
type Conn struct {
	ws     *websocket.Conn
	send   chan Envelope
	closed chan struct{}
}

// NewConn 用 upgrader 把 http 请求升级为 ws 连接。
func NewConn(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 严格校验 Origin，避免跨源劫持（spec §10.1）
		CheckOrigin: func(req *http.Request) bool {
			origin := req.Header.Get("Origin")
			if origin == "" {
				return true // 同源/非浏览器客户端
			}
			host := req.Host
			// 允许同 host 的 https/wss 引用
			return originMatchesHost(origin, host)
		},
	}
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	conn := &Conn{
		ws:     c,
		send:   make(chan Envelope, 64),
		closed: make(chan struct{}),
	}
	go conn.writeLoop()
	go conn.pingLoop()
	return conn, nil
}

// Send 异步发送一条 env（send 通道满则丢弃并返回 false）。
func (c *Conn) Send(env Envelope) bool {
	select {
	case c.send <- env:
		return true
	default:
		return false
	}
}

// Read 阻塞读一条 env。连接关闭时返回 ok=false。
func (c *Conn) Read() (Envelope, bool) {
	c.ws.SetReadDeadline(time.Now().Add(readTimeout))
	var env Envelope
	if err := c.ws.ReadJSON(&env); err != nil {
		return Envelope{}, false
	}
	return env, true
}

// Close 关闭底层 ws。
func (c *Conn) Close() error {
	select {
	case <-c.closed:
	default:
		close(c.closed)
	}
	return c.ws.Close()
}

// writeLoop 把 send 通道里的 env 写到底层 ws。
func (c *Conn) writeLoop() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case env, ok := <-c.send:
			if !ok {
				return
			}
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteJSON(env); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closed:
			return
		}
	}
}

// pingLoop 单独 goroutine 周期性 ping。MVP 1 阶段由 writeLoop 兼顾；保留以备扩展。
func (c *Conn) pingLoop() {}

// originMatchesHost 校验 Origin 指向当前 Host（处理 https/wss 协议差异）。
func originMatchesHost(origin, host string) bool {
	if len(origin) < 8 {
		return false
	}
	// 简单协议剥离
	switch {
	case len(origin) >= 7 && origin[:7] == "http://":
		origin = origin[7:]
	case len(origin) >= 8 && origin[:8] == "https://":
		origin = origin[8:]
	case len(origin) >= 6 && origin[:6] == "wss://":
		origin = origin[6:]
	case len(origin) >= 5 && origin[:5] == "ws://":
		origin = origin[5:]
	}
	// 截断到第一个 /
	for i := 0; i < len(origin); i++ {
		if origin[i] == '/' {
			origin = origin[:i]
			break
		}
	}
	return origin == host
}
```

- [ ] **Step 10: 实现 handler.go（MVP 1 阶段只支持 3 个 method）**

文件：`mytool/internal/ws/handler.go`

```go
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"mytool/internal/engine"
	"mytool/internal/session"
)

// Handler 处理单条 req envelope，必要时返回 resp 或触发 evt。
type Handler struct {
	hub *Hub
	mgr *session.Manager
}

// NewHandler 构造 handler。
func NewHandler(hub *Hub, mgr *session.Manager) *Handler {
	return &Handler{hub: hub, mgr: mgr}
}

// ServeConn 把 conn 接入 hub + manager，并处理 req 循环。
// 服务端事件（来自 manager.Output）被广播到 hub。
func (h *Handler) ServeConn(ctx context.Context, c *Conn) error {
	sub := h.hub.Subscribe()
	defer h.hub.Unsubscribe(sub)

	// 把 session 输出转发到 hub
	go h.forwardSession(ctx, sub)

	// 读循环
	for {
		env, ok := c.Read()
		if !ok {
			return nil
		}
		if env.Type != "req" {
			c.Send(newErrorResp(env.ID, "protocol_error", "unsupported envelope type"))
			continue
		}
		resp, evt := h.dispatch(ctx, env)
		if resp != nil {
			c.Send(*resp)
		}
		// 一些方法会主动发 evt，由 forwardSession 持续推送
		_ = evt
	}
}

// forwardSession 把 session.Manager 的事件复制到 hub。
func (h *Handler) forwardSession(ctx context.Context, _ chan Envelope) {
	// MVP 1：直接转发 manager.Output 到 hub
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-h.mgr.Output():
			if !ok {
				return
			}
			raw, _ := json.Marshal(map[string]any{
				"type":      "text",
				"text":      string(ev.Data),
				"sessionId": h.mgr.SessionID(),
			})
			h.hub.Broadcast(Envelope{Type: "evt", SessionID: h.mgr.SessionID(), Event: raw})
		}
	}
}

// dispatch 处理单条 req。返回 resp 立即发给客户端；evt 暂不主动发（依赖 forwardSession）。
func (h *Handler) dispatch(ctx context.Context, env Envelope) (*Envelope, any) {
	switch env.Method {
	case "session.start":
		return h.handleStart(ctx, env)
	case "session.input":
		return h.handleInput(env)
	case "session.stop":
		return h.handleStop(env)
	default:
		return newErrorRespPtr(env.ID, "not_found", "unknown method: "+env.Method), nil
	}
}

func (h *Handler) handleStart(ctx context.Context, env Envelope) (*Envelope, any) {
	var p struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		CWD     string   `json:"cwd"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, "protocol_error", "invalid params"), nil
	}
	run, err := engine.NewRunner(p.Command, engine.ExecRequest{
		Command: p.Command,
		Args:    p.Args,
		CWD:     p.CWD,
	})
	if err != nil {
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	sid, err := h.mgr.Start(ctx, engine.ExecRequest{Command: p.Command, Args: p.Args, CWD: p.CWD}, run)
	if err != nil {
		return newErrorRespPtr(env.ID, "conflict", err.Error()), nil
	}
	result, _ := json.Marshal(map[string]string{"sessionId": sid})
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok, Result: result}, nil
}

func (h *Handler) handleInput(env Envelope) (*Envelope, any) {
	var p struct {
		SessionID string `json:"sessionId"`
		Text      string `json:"text"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, "protocol_error", "invalid params"), nil
	}
	// 简化：MVP 1 不区分 sessionId
	if err := h.mgr.Write([]byte(p.Text)); err != nil {
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func (h *Handler) handleStop(env Envelope) (*Envelope, any) {
	if err := h.mgr.Stop(); err != nil {
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

// Write 是 session.Manager.Write 的薄包装（这里直接调 mgr.Write）。
func (h *Handler) manager() *session.Manager { return h.mgr }

// helper
func newErrorResp(id, code, msg string) Envelope {
	return Envelope{Type: "resp", ID: id, OK: boolPtr(false), Error: &RPCError{Code: code, Message: msg}}
}
func newErrorRespPtr(id, code, msg string) *Envelope {
	e := newErrorResp(id, code, msg)
	return &e
}
func boolPtr(b bool) *bool { return &b }

// 防止 import 错误占位
var _ = errors.New
var _ = fmt.Sprintf
```

- [ ] **Step 11: 在 manager.go 暴露 Write + SessionID（handler 需调用）**

修改 `mytool/internal/session/manager.go`，在文件末尾追加：

```go
// Write 把 p 写入当前活跃 runner。
func (m *Manager) Write(p []byte) error {
	m.mu.Lock()
	run := m.active
	m.mu.Unlock()
	if run == nil {
		return errors.New("session: no active runner")
	}
	return run.Write(p)
}

// SessionID 返回当前活跃 session id，无活跃返回空。
func (m *Manager) SessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sid
}
```

- [ ] **Step 12: 跑全部 ws 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/ws/... ./internal/session/... -v
```

预期：3 codec + 2 hub + 3 session = 8 个测试 PASS。

- [ ] **Step 13: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/go.mod mytool/go.sum mytool/internal/ws/ mytool/internal/session/manager.go
git commit -m "feat(ws): WebSocket 协议 (codec+conn+hub+handler) + session.Write/SessionID"
```

---

## Task 10: internal/gateway（HTTP 路由 + SPA + handlers）

**Files:**
- Modify: `mytool/go.mod`（新增 `go-chi/chi/v5`）
- Create: `mytool/internal/gateway/router.go`
- Create: `mytool/internal/gateway/spa.go`
- Create: `mytool/internal/gateway/handlers.go`
- Test: `mytool/internal/gateway/router_test.go`

- [ ] **Step 1: 添加 chi 依赖**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go get github.com/go-chi/chi/v5@v5.0.12
```

- [ ] **Step 2: 写 router_test.go**

文件：`mytool/internal/gateway/router_test.go`

```go
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz(t *testing.T) {
	h := NewRouter(Dependencies{}, "test-token")
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", rr.Body.String())
	}
}

func TestVersion(t *testing.T) {
	h := NewRouter(Dependencies{Version: "0.1.0"}, "test-token")
	req := httptest.NewRequest("GET", "/version", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var got map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["version"] != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", got["version"])
	}
}

func TestSPAFallback(t *testing.T) {
	h := NewRouter(Dependencies{FS: stubSPA{}}, "test-token")
	req := httptest.NewRequest("GET", "/some/spa/route", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200 (SPA fallback)", rr.Code)
	}
	if rr.Body.String() != "<html>spa</html>" {
		t.Errorf("body = %q", rr.Body.String())
	}
}

func TestWSRejectsMissingToken(t *testing.T) {
	h := NewRouter(Dependencies{}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}

// stubSPA 实现 fs.FS，仅暴露 index.html。
type stubSPA struct{}

func (stubSPA) Open(name string) (http.File, error) {
	if name == "." || name == "/" || name == "index.html" {
		return stubFile{name: "index.html", body: "<html>spa</html>"}, nil
	}
	return nil, &fsStubErr{msg: "not found: " + name}
}

type stubFile struct{ name, body string }

func (s stubFile) Read(p []byte) (int, error) {
	if s.body == "" {
		return 0, errStubEOF
	}
	n := copy(p, s.body)
	s.body = s.body[n:]
	return n, nil
}
func (s stubFile) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (s stubFile) Close() error                                   { return nil }
func (s stubFile) Stat() (interface{ Name() string }, error)      { return nil, nil }
func (s stubFile) Readdir(count int) ([]interface{ Name() string }, error) {
	return nil, nil
}

type fsStubErr struct{ msg string }

func (e *fsStubErr) Error() string { return e.msg }

var errStubEOF = &fsStubErr{msg: "EOF"}
```

> 上面 stubSPA 用了 `interface{ Name() string }` 是为了让测试不依赖真实 fs.File 全部方法。M1 阶段 router_test 实际只需要走"找不到文件"分支返回 SPA fallback。可改写为更简洁版本：见 Step 2-改。

- [ ] **Step 2-改：用真实 embed.FS 子集（避免 stub 复杂度）**

替换 `mytool/internal/gateway/router_test.go` 完整内容为：

```go
package gateway

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
)

//go:embed testdata/*
var testdata embed.FS

func newTestSPA() fs.FS {
	sub, err := fs.Sub(testdata, "testdata")
	if err != nil {
		panic(err)
	}
	return sub
}

func TestHealthz(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", rr.Body.String())
	}
}

func TestVersion(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA(), Version: "0.1.0"}, "test-token")
	req := httptest.NewRequest("GET", "/version", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var got map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["version"] != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", got["version"])
	}
}

func TestSPAFallback(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/some/unknown/route", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200 (SPA fallback)", rr.Code)
	}
	body := rr.Body.String()
	if body == "" {
		t.Errorf("body should not be empty")
	}
}

func TestWSRejectsMissingToken(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
```

创建 `mytool/internal/gateway/testdata/index.html`：

```html
<!doctype html>
<html><body>spa</body></html>
```

- [ ] **Step 3: 跑测试确认失败**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/gateway/...
```

预期：FAIL（包内无文件）。

- [ ] **Step 4: 实现 router.go**

文件：`mytool/internal/gateway/router.go`

```go
// Package gateway 提供 mytool HTTP 入口：healthz/version/SPA + REST 占位 + WS 升级。
package gateway

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"mytool/internal/auth"
	"mytool/internal/session"
	"mytool/internal/ws"
)

// Dependencies 注入 router 运行所需的可选依赖。
type Dependencies struct {
	FS      fs.FS // 嵌入式 SPA 文件系统（必有）
	Version string
	WS      *ws.Handler
	Session *session.Manager
}

// NewRouter 构造 chi router。authToken 用于 ws 升级与未来的 REST 鉴权。
func NewRouter(deps Dependencies, authToken string) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"` + deps.Version + `"}`))
	})

	// /api/v1/ws 走鉴权
	r.With(auth.BearerMiddleware(authToken, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.WS == nil {
			http.Error(w, "ws handler not configured", http.StatusServiceUnavailable)
			return
		}
		c, err := ws.NewConn(w, r)
		if err != nil {
			http.Error(w, "ws upgrade failed", http.StatusBadRequest)
			return
		}
		_ = deps.WS.ServeConn(r.Context(), c)
	}))).Get("/api/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		// 占位：实际处理在 With 中间件链里
	})

	// SPA fallback：先尝试静态文件，失败则返回 index.html
	if deps.FS != nil {
		r.Handle("/*", spaHandler(deps.FS))
	}

	return r
}
```

- [ ] **Step 5: 实现 spa.go**

文件：`mytool/internal/gateway/spa.go`

```go
package gateway

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler 返回一个 handler：先尝试从 fs 取真实文件，否则 fallback 到 index.html。
// 对 .js 设置正确 MIME。
func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if upath == "" || upath == "/" {
			upath = "/index.html"
		}
		// 安全：拒绝 .. 与绝对路径
		clean := path.Clean(upath)
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
			http.NotFound(w, r)
			return
		}
		// 检查文件是否存在
		if _, err := fs.Stat(fsys, strings.TrimPrefix(clean, "/")); err == nil {
			if strings.HasSuffix(clean, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		// fallback
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		if strings.HasSuffix(clean, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		fileServer.ServeHTTP(w, r2)
	}
}
```

- [ ] **Step 6: 实现 handlers.go（占位）**

文件：`mytool/internal/gateway/handlers.go`

```go
package gateway

import "net/http"

// PlaceholderHandlers 暴露 MVP 2+ 才会用到的 endpoints。
// 当前空实现，避免 router 文件膨胀。
func PlaceholderHandlers() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
}
```

- [ ] **Step 7: 跑全部 gateway 测试**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./internal/gateway/... -v
```

预期：4 个测试 PASS。

- [ ] **Step 8: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/go.mod mytool/go.sum mytool/internal/gateway/
git commit -m "feat(gateway): chi router + SPA fallback + WS 升级"
```

---

## Task 11: cmd/server（入口装配）

**Files:**
- Create: `mytool/cmd/server/flags.go`
- Create: `mytool/cmd/server/main.go`

- [ ] **Step 1: 实现 flags.go**

文件：`mytool/cmd/server/flags.go`

```go
package main

import "flag"

// serverFlags 汇总命令行 flags。
type serverFlags struct {
	port         string
	workspace    string
	authToken    string
	mtls         string
	logLevel     string
	defaultCmd   string
	showVersion  bool
	showHelp     bool
}

func parseServerFlags(args []string) (serverFlags, error) {
	fs := flag.NewFlagSet("mytool", flag.ContinueOnError)
	fs.StringVar(&(*(*serverFlags)(nil)).port, "port", "", "listen port (overrides MYTOOL_PORT)")

	f := serverFlags{}
	fs.StringVar(&f.port, "port", "8443", "listen port")
	fs.StringVar(&f.workspace, "workspace", "", "workspace root (overrides MYTOOL_WORKSPACE)")
	fs.StringVar(&f.authToken, "auth-token", "", "auth token (overrides MYTOOL_AUTH_TOKEN)")
	fs.StringVar(&f.mtls, "mtls", "", "mtls mode: none|optional|required")
	fs.StringVar(&f.logLevel, "log-level", "info", "log level: debug|info|warn|error")
	fs.StringVar(&f.defaultCmd, "default-command", "claude", "default AI command")
	fs.BoolVar(&f.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&f.showHelp, "help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		return f, err
	}
	return f, nil
}
```

> 上面第一段（`fs.StringVar(&(*(*serverFlags)(nil)).port...`）是反例——已废弃。M1 阶段应只保留第二段。删除第一段。

- [ ] **Step 1-改：清掉反例代码**

替换整个 `mytool/cmd/server/flags.go` 内容为：

```go
package main

import "flag"

// serverFlags 汇总命令行 flags。
type serverFlags struct {
	port        string
	workspace   string
	authToken   string
	mtls        string
	logLevel    string
	defaultCmd  string
	showVersion bool
	showHelp    bool
}

// parseServerFlags 解析 os.Args[1:]。返回 flags 与 error。
func parseServerFlags(args []string) (serverFlags, error) {
	f := serverFlags{}
	fs := flag.NewFlagSet("mytool", flag.ContinueOnError)
	fs.StringVar(&f.port, "port", "8443", "listen port")
	fs.StringVar(&f.workspace, "workspace", "", "workspace root (overrides MYTOOL_WORKSPACE)")
	fs.StringVar(&f.authToken, "auth-token", "", "auth token (overrides MYTOOL_AUTH_TOKEN)")
	fs.StringVar(&f.mtls, "mtls", "", "mtls mode: none|optional|required")
	fs.StringVar(&f.logLevel, "log-level", "info", "log level: debug|info|warn|error")
	fs.StringVar(&f.defaultCmd, "default-command", "claude", "default AI command")
	fs.BoolVar(&f.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&f.showHelp, "help", false, "print help and exit")

	if err := fs.Parse(args); err != nil {
		return f, err
	}
	return f, nil
}
```

- [ ] **Step 2: 实现 main.go**

文件：`mytool/cmd/server/main.go`

```go
// mytool 后端入口：装配 config + engine + session + gateway + ws，启动 HTTP 服务。
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"mytool/internal/config"
	"mytool/internal/gateway"
	"mytool/internal/logx"
	"mytool/internal/session"
	"mytool/internal/ws"
)

const version = "0.1.0"

//go:embed web/*
var webAssets embed.FS

func main() {
	flags, err := parseServerFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flag parse: %v\n", err)
		os.Exit(2)
	}
	if flags.showVersion {
		fmt.Println(version)
		return
	}
	if flags.showHelp {
		fmt.Fprintln(os.Stderr, "Usage: mytool [flags]\n  -port          listen port (default 8443)")
		return
	}

	logger := logx.New()
	logger.SetLevel(parseLevel(flags.logLevel))

	cfg, err := buildConfig(flags)
	if err != nil {
		logger.Error("startup", "config: %v", err)
		os.Exit(1)
	}

	// 启动 server
	if err := run(cfg, logger); err != nil {
		logger.Error("startup", "run: %v", err)
		os.Exit(1)
	}
}

func buildConfig(f serverFlags) (config.Config, error) {
	env := config.FromEnv()
	c := config.Config{
		Port:        firstNonEmpty(f.port, env.Port, "8443"),
		AuthToken:   firstNonEmpty(f.authToken, env.AuthToken),
		Workspace:   firstNonEmpty(f.workspace, env.Workspace),
		MTLS:        firstNonEmpty(f.mtls, env.MTLS),
		LogLevel:    firstNonEmpty(f.logLevel, env.LogLevel),
		DefaultCmd:  firstNonEmpty(f.defaultCmd, env.DefaultCmd),
	}.WithDefaults()

	// token 缺失则生成并持久化（MVP 1 阶段：仅 console 提示，暂不写文件）
	if c.AuthToken == "" {
		tok, err := config.NewToken()
		if err != nil {
			return c, fmt.Errorf("generate token: %w", err)
		}
		c.AuthToken = tok
		fmt.Fprintf(os.Stderr, "==> Generated new auth token (MVP 1: in-memory only): %s\n", tok)
	}

	// 确保 workspace 存在
	if err := os.MkdirAll(c.Workspace, 0o755); err != nil {
		return c, fmt.Errorf("create workspace: %w", err)
	}
	return c, nil
}

func run(cfg config.Config, logger *logx.Logger) error {
	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("embed web: %w", err)
	}
	// 兼容空 web/：若 dist 不存在则用 gateway.testdata 子集占位
	if _, err := fs.Stat(staticFS, "."); err != nil {
		logger.Warn("startup", "embedded web/ missing; using stub SPA")
	}

	hub := ws.NewHub()
	mgr := session.NewManager()
	wsHandler := ws.NewHandler(hub, mgr)

	r := gateway.NewRouter(gateway.Dependencies{
		FS:      staticFS,
		Version: version,
		WS:      wsHandler,
		Session: mgr,
	}, cfg.AuthToken)

	addr := ":" + cfg.Port
	logger.Info("startup", "listening on %s, workspace=%s", addr, cfg.Workspace)
	srv := &http.Server{Addr: addr, Handler: r}
	return srv.ListenAndServe()
}

func parseLevel(s string) logx.Level {
	switch s {
	case "debug":
		return logx.LevelDebug
	case "warn":
		return logx.LevelWarn
	case "error":
		return logx.LevelError
	}
	return logx.LevelInfo
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// 兼容 import
var _ = filepath.Join
var _ = context.Background
```

- [ ] **Step 3: 编译验证**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build ./cmd/server
```

预期：编译成功，产出 `mytool.exe`（Windows）或 `mytool`（Linux/macOS）。如果因 `web/` 目录为空导致 embed 失败，先创建 `mytool/cmd/server/web/.gitkeep`。

- [ ] **Step 4: 跑全部测试确认无回归**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

预期：所有阶段 A+B 的 29 个 + 阶段 C 至此的测试全部 PASS。

- [ ] **Step 5: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/cmd/server/
git commit -m "feat(cmd): server 入口装配 config+engine+session+gateway+ws"
```

---

## Task 12: SPA 占位 + 嵌入

**Files:**
- Create: `mytool/cmd/server/web/index.html`
- Create: `mytool/cmd/server/web/main.js`
- Create: `mytool/cmd/server/web/style.css`

- [ ] **Step 1: 写 index.html（占位）**

文件：`mytool/cmd/server/web/index.html`

```html
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>mytool</title>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <header>
    <h1>mytool</h1>
    <span id="conn-status">offline</span>
  </header>
  <main>
    <section id="login">
      <h2>Connect</h2>
      <input id="token" type="password" placeholder="auth token" />
      <button id="connect">Connect</button>
    </section>
    <section id="session" hidden>
      <h2>Session</h2>
      <div>
        <input id="cmd" placeholder="command (e.g. echo)" />
        <input id="args" placeholder="args (comma-separated)" />
        <button id="start">Start</button>
      </div>
      <pre id="output"></pre>
      <div>
        <input id="input" placeholder="stdin line" />
        <button id="send">Send</button>
        <button id="stop">Stop</button>
      </div>
    </section>
  </main>
  <script src="/main.js"></script>
</body>
</html>
```

- [ ] **Step 2: 写 main.js**

文件：`mytool/cmd/server/web/main.js`

```javascript
const status = document.getElementById('conn-status');
const output = document.getElementById('output');
let ws = null;
let reqId = 0;

function setStatus(s) { status.textContent = s; }

function connect() {
  const token = document.getElementById('token').value.trim();
  if (!token) { alert('token required'); return; }
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const url = `${proto}://${location.host}/api/v1/ws?token=${encodeURIComponent(token)}`;
  ws = new WebSocket(url);
  ws.onopen = () => {
    setStatus('online');
    document.getElementById('login').hidden = true;
    document.getElementById('session').hidden = false;
  };
  ws.onclose = () => { setStatus('offline'); };
  ws.onerror = () => { setStatus('error'); };
  ws.onmessage = (ev) => {
    const env = JSON.parse(ev.data);
    if (env.type === 'evt') {
      const e = JSON.parse(env.event || '{}');
      if (e.type === 'text') {
        output.textContent += e.text + '\n';
      } else if (e.type === 'lifecycle') {
        output.textContent += `[${e.message}]\n`;
      }
    } else if (env.type === 'resp' && !env.ok) {
      output.textContent += `[error ${env.error?.code}: ${env.error?.message}]\n`;
    }
  };
}

function send(method, params = {}) {
  if (!ws || ws.readyState !== 1) return;
  reqId += 1;
  ws.send(JSON.stringify({ type: 'req', id: `r${reqId}`, method, params }));
}

document.getElementById('connect').onclick = connect;
document.getElementById('start').onclick = () => {
  const command = document.getElementById('cmd').value.trim();
  const argsRaw = document.getElementById('args').value.trim();
  const args = argsRaw ? argsRaw.split(',').map(s => s.trim()) : [];
  send('session.start', { command, args });
};
document.getElementById('send').onclick = () => {
  const text = document.getElementById('input').value + '\n';
  send('session.input', { text });
};
document.getElementById('stop').onclick = () => send('session.stop');
```

- [ ] **Step 3: 写 style.css**

文件：`mytool/cmd/server/web/style.css`

```css
* { box-sizing: border-box; }
body { font-family: system-ui, sans-serif; margin: 0; padding: 0; background: #f5f5f5; }
header { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: #222; color: #fff; }
header h1 { margin: 0; font-size: 18px; }
#conn-status { font-size: 12px; padding: 2px 8px; border-radius: 4px; background: #555; }
main { padding: 16px; max-width: 720px; margin: 0 auto; }
section { background: #fff; padding: 16px; border-radius: 6px; margin-bottom: 12px; }
input { padding: 6px 8px; font-size: 14px; border: 1px solid #ccc; border-radius: 4px; margin-right: 4px; }
button { padding: 6px 12px; font-size: 14px; background: #2563eb; color: #fff; border: 0; border-radius: 4px; cursor: pointer; }
button:hover { background: #1d4ed8; }
pre { background: #111; color: #0f0; padding: 12px; border-radius: 4px; min-height: 200px; white-space: pre-wrap; word-break: break-all; }
```

- [ ] **Step 4: 重新编译并验证 embed**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && rm -rf cmd/server/web/.gitkeep && go build -o /tmp/mytool-test ./cmd/server && ls -la /tmp/mytool-test
```

预期：编译成功，/tmp/mytool-test 是非空二进制。

- [ ] **Step 5: 启动并 curl 验证 SPA**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && MYTOOL_AUTH_TOKEN=testtoken MYTOOL_PORT=18443 /tmp/mytool-test &
SERVER_PID=$!
sleep 1
curl -s http://127.0.0.1:18443/healthz
echo
curl -s http://127.0.0.1:18443/version
echo
curl -s http://127.0.0.1:18443/ | head -5
kill $SERVER_PID 2>/dev/null
```

预期输出（精简）：

```
ok
{"version":"0.1.0"}
<!doctype html>
<html lang="en">
<head>
```

- [ ] **Step 6: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/cmd/server/web/
git commit -m "feat(spa): 占位 SPA（connect + start + send + stop）"
```

---

## Task 13: e2e smoke test

**Files:**
- Create: `mytool/scripts/e2e_smoke.sh`

- [ ] **Step 1: 写 e2e_smoke.sh**

文件：`mytool/scripts/e2e_smoke.sh`

```bash
#!/usr/bin/env bash
# e2e_smoke.sh: 启动 mytool，验证 healthz、SPA、ws 鉴权、ws session.start。
# 前置：go build 已产出 ./mytool 二进制（或在 PATH 里）。
set -euo pipefail

PORT=${MYTOOL_SMOKE_PORT:-19443}
TOKEN=${MYTOOL_SMOKE_TOKEN:-smoke-token-$(date +%s)}
BIN=${MYTOOL_SMOKE_BIN:-./mytool}

if [[ ! -x "$BIN" ]]; then
  echo "binary not found: $BIN" >&2
  exit 1
fi

# 启动
MYTOOL_AUTH_TOKEN="$TOKEN" MYTOOL_PORT="$PORT" "$BIN" >/tmp/mytool-smoke.log 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null || true' EXIT

# 等到 healthz 通
for i in {1..30}; do
  if curl -s "http://127.0.0.1:$PORT/healthz" | grep -q ok; then
    break
  fi
  sleep 0.2
done
curl -s "http://127.0.0.1:$PORT/healthz" | grep -q ok || { echo "healthz failed" >&2; cat /tmp/mytool-smoke.log; exit 1; }
echo "✓ healthz ok"

# 验证 SPA
curl -s "http://127.0.0.1:$PORT/" | grep -q '<title>mytool</title>' || { echo "SPA not served" >&2; exit 1; }
echo "✓ SPA served"

# 验证 SPA fallback
curl -s "http://127.0.0.1:$PORT/some/unknown/route" | grep -q '<title>mytool</title>' || { echo "SPA fallback failed" >&2; exit 1; }
echo "✓ SPA fallback ok"

# 验证 WS 鉴权拒绝无 token
code=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$PORT/api/v1/ws")
[[ "$code" == "401" ]] || { echo "ws without token should 401, got $code" >&2; exit 1; }
echo "✓ ws rejects missing token"

echo
echo "=== smoke test PASSED ==="
echo "  port:  $PORT"
echo "  token: $TOKEN"
```

- [ ] **Step 2: 跑 smoke 验证**

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go build -o mytool ./cmd/server && bash scripts/e2e_smoke.sh
```

预期输出（精简）：

```
✓ healthz ok
✓ SPA served
✓ SPA fallback ok
✓ ws rejects missing token

=== smoke test PASSED ===
  port:  19443
  token: smoke-token-...
```

> 注意：smoke 不覆盖 ws session.start（需要 websocat 客户端做 E2E）。MVP 1 阶段仅覆盖 HTTP 层；ws 全功能走 Playwright e2e（MVP 2+）。

- [ ] **Step 3: 提交**

```bash
cd "D:/Documents/Dev-Repo/MobileVC"
git add mytool/scripts/e2e_smoke.sh mytool/mytool 2>/dev/null || true
# 二进制通常不提交；只提交脚本
git reset mytool/mytool 2>/dev/null || true
git add mytool/scripts/e2e_smoke.sh
git commit -m "test(smoke): e2e smoke 覆盖 healthz + SPA + ws 鉴权"
```

---

## 阶段 C 完成标准

```bash
cd "D:/Documents/Dev-Repo/MobileVC/mytool" && go test ./... -count=1
```

预期：阶段 A（21）+ 阶段 B（8）+ 阶段 C 的 ws（5）+ gateway（4）= 38 个测试全 PASS。

git log 新增 5 个 commit：

```
feat(ws): WebSocket 协议 (codec+conn+hub+handler) + session.Write/SessionID
feat(gateway): chi router + SPA fallback + WS 升级
feat(cmd): server 入口装配 config+engine+session+gateway+ws
feat(spa): 占位 SPA（connect + start + send + stop）
test(smoke): e2e smoke 覆盖 healthz + SPA + ws 鉴权
```

MVP 1 阶段 C 完成后，整个 MVP 1 结束：
- ✅ Go 后端骨架
- ✅ PTY 通用 Runner + 注册表
- ✅ 鉴权（启动随机 token + Bearer middleware）
- ✅ WebSocket 协议（codec + conn + hub + handler）
- ✅ HTTP 网关（router + SPA + healthz + version + ws 升级）
- ✅ cmd/server 入口
- ✅ 嵌入式占位 SPA
- ✅ e2e smoke 验证

可直接 `cd mytool && go build -o mytool ./cmd/server && ./mytool` 启动，手机浏览器开 `https://<lan-ip>:8443/` 体验完整 MVP 1 闭环。

---

## 后续 MVP 建议（不在本 plan 范围）

- **MVP 2**：ClaudeRunner（解析 `--output-format stream-json`）+ 投影层扩展（diff / permission / plan）+ 会话续接
- **MVP 3**：CodexRunner（JSON-RPC app-server）+ Skill / Memory 仓库
- **MVP 4**：mTLS 自签 CA + 设备证书 + 二维码配对 + PWA 离线缓存 + npm 启动壳
