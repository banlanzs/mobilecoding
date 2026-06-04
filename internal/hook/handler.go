// Package hook 实现 Claude Code v2.1+ HTTP hooks 端点。
// Claude CLI 在 PermissionRequest 事件触发时 POST 到 /v1/hooks/permission-request，
// 本包负责把请求转发给 WebSocket 客户端（手机端），等用户决策后返回给 Claude。
package hook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Request 是 Claude CLI 发来的 HTTP hook 请求体。
// 文档参考：https://docs.claude.com/en/docs/claude-code/hooks
type Request struct {
	SessionID      string          `json:"session_id"`
	TranscriptPath string          `json:"transcript_path"`
	CWD            string          `json:"cwd"`
	HookEventName  string          `json:"hook_event_name"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
}

// Decision 是手机端对权限请求的回应。
type Decision struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason,omitempty"`
}

// Response 是返回给 Claude CLI 的 JSON。
// 与 Claude 文档一致：必须用 2xx + JSON body 表达 allow/deny，HTTP 状态码无法阻断。
type Response struct {
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput"`
}

type HookSpecificOutput struct {
	HookEventName string  `json:"hookEventName"`
	Decision      Decision2 `json:"decision"`
}

type Decision2 struct {
	Behavior string `json:"behavior"` // "allow" | "deny"
	Message  string `json:"message,omitempty"`
}

// Event 是注册到 WebSocket 上的事件载荷（手机端订阅的契约）。
type Event struct {
	Type            string          `json:"type"` // 固定 "permission_request"
	RequestID       string          `json:"requestId"`
	HookEventName   string          `json:"hookEventName"`
	ToolName        string          `json:"toolName"`
	ToolInput       json.RawMessage `json:"toolInput"`
	ToolInputPrompt string          `json:"toolInputPrompt"` // 人类可读摘要
	SessionID       string          `json:"sessionId"`
	CWD             string          `json:"cwd"`
	IssuedAt        time.Time       `json:"issuedAt"`
}

// DefaultTimeout 是单个权限请求等待用户响应的默认上限。
const DefaultTimeout = 5 * time.Minute

// DefaultDenyOnTimeout 控制超时后默认决策。true = 拒绝（更安全），false = 允许。
const DefaultDenyOnTimeout = true

// Registry 管理当前等待响应的权限请求。
// 每个 requestId 对应一个 channel，HTTP handler 通过它阻塞等响应，
// WebSocket handler 调用 Respond 写入决策并唤醒 HTTP handler。
type Registry struct {
	mu      sync.Mutex
	pending map[string]chan Decision
}

func NewRegistry() *Registry {
	return &Registry{pending: make(map[string]chan Decision)}
}

// Register 创建一个新的待响应项，返回 (requestId, channel)。
// HTTP handler 拿到 channel 后阻塞等待 Respond。
func (r *Registry) Register() (string, <-chan Decision) {
	id := "permreq_" + uuid.NewString()
	ch := make(chan Decision, 1)
	r.mu.Lock()
	r.pending[id] = ch
	r.mu.Unlock()
	return id, ch
}

// Respond 写入决策并关闭 channel，唤醒等待的 HTTP handler。
// 重复响应或响应未知 id 会被忽略。
func (r *Registry) Respond(requestID string, d Decision) bool {
	r.mu.Lock()
	ch, ok := r.pending[requestID]
	if ok {
		delete(r.pending, requestID)
	}
	r.mu.Unlock()
	if !ok {
		return false
	}
	ch <- d
	close(ch)
	return true
}

// Abort 取消等待（用于 server 关闭）。
func (r *Registry) Abort(requestID string) {
	r.mu.Lock()
	ch, ok := r.pending[requestID]
	if ok {
		delete(r.pending, requestID)
	}
	r.mu.Unlock()
	if ok {
		close(ch)
	}
}

// Pending 返回当前待响应数量（用于调试 / 监控）。
func (r *Registry) Pending() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.pending)
}

// Handler 返回 HTTP handler，注册到 chi router 的 /v1/hooks/permission-request。
// 接收 Claude POST -> 构造 Event -> 调用 broadcaster -> 等待用户决策 -> 返回 JSON。
type Handler struct {
	Registry      *Registry
	Broadcast     func(Event) // 由 main.go 注入：把事件广播到所有 WS 订阅者
	Timeout       time.Duration
	DenyOnTimeout bool
	Log           func(format string, args ...any) // 可选日志回调
}

func NewHandler(reg *Registry, broadcast func(Event)) *Handler {
	return &Handler{
		Registry:      reg,
		Broadcast:     broadcast,
		Timeout:       DefaultTimeout,
		DenyOnTimeout: DefaultDenyOnTimeout,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logf("hook received request: method=%s remote=%s", r.Method, r.RemoteAddr)

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析 Claude POST body
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logf("hook parse error: %v", err)
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	h.logf("hook parsed: tool=%s event=%s session=%s", req.ToolName, req.HookEventName, req.SessionID)
	if req.HookEventName == "" {
		req.HookEventName = "PermissionRequest"
	}
	if req.ToolName == "" {
		http.Error(w, "tool_name is required", http.StatusBadRequest)
		return
	}

	// 摘要
	prompt := summarizeToolInput(req.ToolName, req.ToolInput)

	// 注册并构造事件
	requestID, ch := h.Registry.Register()
	ev := Event{
		Type:            "permission_request",
		RequestID:       requestID,
		HookEventName:   req.HookEventName,
		ToolName:        req.ToolName,
		ToolInput:       req.ToolInput,
		ToolInputPrompt: prompt,
		SessionID:       req.SessionID,
		CWD:             req.CWD,
		IssuedAt:        time.Now().UTC(),
	}

	// 广播给 WS 客户端
	if h.Broadcast != nil {
		h.logf("hook broadcasting to WS: requestId=%s tool=%s", requestID, req.ToolName)
		h.Broadcast(ev)
	}

	// 等待用户决策或超时
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	var decision Decision
	select {
	case d, ok := <-ch:
		if !ok {
			// 通道被关闭（Abort / 异常）→ 默认拒绝
			decision = Decision{Allow: false, Reason: "registry aborted"}
		} else {
			decision = d
			h.logf("hook user responded: requestId=%s allow=%v", requestID, d.Allow)
		}
	case <-ctx.Done():
		h.Registry.Abort(requestID)
		if h.DenyOnTimeout {
			decision = Decision{Allow: false, Reason: "timeout (no user response)"}
		} else {
			decision = Decision{Allow: true, Reason: "timeout (auto-allowed)"}
		}
		h.logf("hook decision timed out: requestId=%s allow=%v", requestID, decision.Allow)
	}

	// 构造 Claude 期望的响应
	resp := Response{
		HookSpecificOutput: HookSpecificOutput{
			HookEventName: req.HookEventName,
			Decision: Decision2{
				Behavior: "allow",
			},
		},
	}
	if !decision.Allow {
		resp.HookSpecificOutput.Decision.Behavior = "deny"
		resp.HookSpecificOutput.Decision.Message = decision.Reason
		if resp.HookSpecificOutput.Decision.Message == "" {
			resp.HookSpecificOutput.Decision.Message = "Denied by user"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// summarizeToolInput 为常见工具生成人类可读摘要。
func summarizeToolInput(toolName string, raw json.RawMessage) string {
	if len(raw) == 0 {
		return toolName
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return toolName
	}
	switch toolName {
	case "Bash":
		if cmd, ok := m["command"].(string); ok {
			if desc, ok := m["description"].(string); ok && desc != "" {
				return fmt.Sprintf("Bash: %s\n  $ %s", desc, truncate(cmd, 240))
			}
			return fmt.Sprintf("Bash: $ %s", truncate(cmd, 300))
		}
	case "Write":
		if p, ok := m["file_path"].(string); ok {
			return fmt.Sprintf("Write: %s", p)
		}
	case "Edit", "MultiEdit":
		if p, ok := m["file_path"].(string); ok {
			return fmt.Sprintf("%s: %s", toolName, p)
		}
	case "Read":
		if p, ok := m["file_path"].(string); ok {
			return fmt.Sprintf("Read: %s", p)
		}
	}
	// fallback：工具名 + 简短 JSON
	b, _ := json.Marshal(m)
	return fmt.Sprintf("%s: %s", toolName, truncate(string(b), 200))
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (h *Handler) logf(format string, args ...any) {
	if h.Log != nil {
		h.Log(format, args...)
	}
}

// errNoBroadcast 哨兵错误。
var errNoBroadcast = errors.New("broadcast not configured")
var _ = errNoBroadcast
