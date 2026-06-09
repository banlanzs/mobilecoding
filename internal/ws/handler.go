package ws

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/banlanzs/mobilecoding/internal/engine"
	"github.com/banlanzs/mobilecoding/internal/hook"
	"github.com/banlanzs/mobilecoding/internal/logx"
	"github.com/banlanzs/mobilecoding/internal/projection"
	"github.com/banlanzs/mobilecoding/internal/protocol"
	"github.com/banlanzs/mobilecoding/internal/session"
)

type Handler struct {
	hub             *Hub
	mgr             *session.Manager
	logger          *logx.Logger
	hookRegistry    *hook.Registry // 可选：用于 permission.respond
	mu              sync.Mutex
	pendingResumeID string // 待使用的 Claude resume session ID
}

// SetPendingResumeID 设置待使用的 resume session ID。
// 下次 session.start 时会自动传递给 ClaudeRunner。
func (h *Handler) SetPendingResumeID(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingResumeID = id
}

func (h *Handler) consumePendingResumeID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	id := h.pendingResumeID
	h.pendingResumeID = ""
	return id
}

func NewHandler(hub *Hub, mgr *session.Manager, logger *logx.Logger) *Handler {
	return &Handler{hub: hub, mgr: mgr, logger: logger}
}

// SetHookRegistry 注入 hook.Registry（用于接收手机端 permission.respond）。
func (h *Handler) SetHookRegistry(reg *hook.Registry) {
	h.hookRegistry = reg
}

// SubscriberCount 返回当前 WebSocket 订阅者数量。
func (h *Handler) SubscriberCount() int {
	return h.hub.SubscriberCount()
}

func (h *Handler) ServeConn(ctx context.Context, c *Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 订阅 Hub 广播
	subCh := h.hub.Subscribe()
	defer h.hub.Unsubscribe(subCh)

	// 桥接协程：subCh → c.send（writeLoop 独占 WebSocket）
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-subCh:
				if !ok {
					return
				}
				select {
				case c.send <- env:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// 阻塞发送辅助函数（发送响应到当前连接）
	sendResp := func(env Envelope) {
		select {
		case c.send <- env:
		case <-ctx.Done():
		}
	}

	for {
		env, ok := c.Read()
		if !ok {
			h.logger.Debug("session", "client disconnected")
			cancel()
			<-done
			return nil
		}
		if env.Type != "req" {
			sendResp(newErrorResp(env.ID, protocol.ErrProtocolError, "unsupported envelope type"))
			continue
		}
		resp, evt := h.dispatch(ctx, env)
		if resp != nil {
			sendResp(*resp)
		}
		_ = evt
	}
}

func (h *Handler) forwardSession(ctx context.Context, out chan<- Envelope) {
	defer close(out)

	input := h.mgr.Output()
	fwdCount := 0

	for {
		select {
		case ev, ok := <-input:
			if !ok {
				h.logger.Debug("session", "forwardSession: input closed, forwarded %d envelopes", fwdCount)
				return
			}
			sid := h.mgr.SessionID()
			projEvents := projection.Project([]engine.Event{ev}, sid)
			h.logger.Debug("session", "forwardSession: event kind=%s projected=%d", ev.Kind, len(projEvents))
			for _, pe := range projEvents {
				env, err := projectionToEnvelope(pe)
				if err != nil {
					h.logger.Error("session", "forwardSession: projectionToEnvelope failed: %v", err)
					continue
				}
				select {
				case out <- env:
					fwdCount++
					h.logger.Debug("session", "forwardSession: sent envelope #%d type=%s", fwdCount, pe.Type)
				case <-ctx.Done():
					h.logger.Debug("session", "forwardSession: context cancelled after %d envelopes", fwdCount)
					return
				}
			}
		case <-ctx.Done():
			h.logger.Debug("session", "forwardSession: context cancelled, forwarded %d envelopes", fwdCount)
			return
		}
	}
}

func (h *Handler) dispatch(ctx context.Context, env Envelope) (*Envelope, any) {
	switch env.Method {
	case protocol.MethodSessionStart:
		return h.handleStart(ctx, env)
	case protocol.MethodSessionInput:
		return h.handleInput(env)
	case protocol.MethodSessionStop:
		return h.handleStop(env)
	case protocol.MethodSessionAbort:
		return h.handleAbort(env)
	case protocol.MethodSessionPermissionAnswer:
		return h.handlePermissionAnswer(env)
	case protocol.MethodPermissionRespond:
		return h.handlePermissionRespond(env)
	default:
		return newErrorRespPtr(env.ID, protocol.ErrNotFound, "unknown method: "+env.Method), nil
	}
}

func (h *Handler) handleStart(ctx context.Context, env Envelope) (*Envelope, any) {
	var p struct {
		Command         string   `json:"command"`
		Args            []string `json:"args"`
		CWD             string   `json:"cwd"`
		ResumeSessionID string   `json:"resumeSessionId"`
		Restart         bool     `json:"restart"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, protocol.ErrProtocolError, "invalid params"), nil
	}

	// 如果请求中没有 resume ID，检查是否有待使用的 pending resume ID（mc CLI 设置的）
	if p.ResumeSessionID == "" {
		p.ResumeSessionID = h.consumePendingResumeID()
	}

	h.logger.Info("session", "starting: command=%s args=%v cwd=%s resumeId=%s", p.Command, p.Args, p.CWD, p.ResumeSessionID)

	req := engine.ExecRequest{
		Command:         p.Command,
		Args:            p.Args,
		CWD:             p.CWD,
		VisibleTerminal: false,
		ResumeSessionID: p.ResumeSessionID,
	}
	run, err := engine.NewRunner(p.Command, req)
	if err != nil {
		h.logger.Error("session", "new runner failed: command=%s err=%v", p.Command, err)
		return newErrorRespPtr(env.ID, protocol.ErrEngineFailure, err.Error()), nil
	}
	var sid string
	if p.Restart {
		sid, err = h.mgr.Restart(ctx, req, run)
	} else {
		sid, err = h.mgr.Start(ctx, req, run)
	}
	if err != nil {
		h.logger.Error("session", "start failed: command=%s restart=%v err=%v", p.Command, p.Restart, err)
		return newErrorRespPtr(env.ID, protocol.ErrConflict, err.Error()), nil
	}
	result, _ := json.Marshal(map[string]string{"sessionId": sid})
	ok := true
	h.logger.Info("session", "started: command=%s sessionId=%s resumeId=%s restart=%v", p.Command, sid, p.ResumeSessionID, p.Restart)
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok, Result: result}, nil
}

func (h *Handler) handleInput(env Envelope) (*Envelope, any) {
	var p struct {
		SessionID string `json:"sessionId"`
		Text      string `json:"text"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, protocol.ErrProtocolError, "invalid params"), nil
	}
	h.logger.Debug("session", "input: sessionId=%s len=%d", p.SessionID, len(p.Text))
	if err := h.mgr.Write([]byte(p.Text + "\n")); err != nil {
		h.logger.Error("session", "write input failed: err=%v", err)
		return newErrorRespPtr(env.ID, protocol.ErrEngineFailure, err.Error()), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func (h *Handler) handleAbort(env Envelope) (*Envelope, any) {
	h.logger.Info("session", "aborting current turn")
	h.mgr.Abort()
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func (h *Handler) handlePermissionAnswer(env Envelope) (*Envelope, any) {
	var p struct {
		Allow     bool   `json:"allow"`
		ToolName  string `json:"toolName"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, protocol.ErrProtocolError, "invalid params"), nil
	}
	h.logger.Info("session", "permission answer: tool=%s allow=%v requestId=%s", p.ToolName, p.Allow, p.RequestID)

	// 优先使用 control_response 协议（Claude stdio permission tool 的标准格式）
	var payload []byte
	if p.RequestID != "" {
		payload = engine.ControlResponse(p.RequestID, p.Allow)
	} else {
		payload = engine.PermissionAnswer(p.Allow, p.ToolName)
	}

	if err := h.mgr.SendToStdin(payload); err != nil {
		h.logger.Error("session", "permission answer write failed: %v", err)
		return newErrorRespPtr(env.ID, protocol.ErrEngineFailure, err.Error()), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func (h *Handler) handleStop(env Envelope) (*Envelope, any) {
	if err := h.mgr.Stop(); err != nil {
		return newErrorRespPtr(env.ID, protocol.ErrEngineFailure, err.Error()), nil
	}
	h.logger.Info("session", "stopped")
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

// handlePermissionRespond 接收手机端对 HTTP hook 权限请求的回应。
// 这是新协议（permission.respond），与 session.permission.answer（控制 stdin 旧协议）并存。
func (h *Handler) handlePermissionRespond(env Envelope) (*Envelope, any) {
	if h.hookRegistry == nil {
		return newErrorRespPtr(env.ID, protocol.ErrNotConfigured, "hook registry not configured"), nil
	}
	var p struct {
		RequestID string `json:"requestId"`
		Allow     bool   `json:"allow"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal(env.Params, &p); err != nil {
		return newErrorRespPtr(env.ID, protocol.ErrProtocolError, "invalid params"), nil
	}
	if p.RequestID == "" {
		return newErrorRespPtr(env.ID, protocol.ErrProtocolError, "requestId required"), nil
	}
	h.logger.Info("session", "permission.respond: id=%s allow=%v reason=%q", p.RequestID, p.Allow, p.Reason)
	if !h.hookRegistry.Respond(p.RequestID, hook.Decision{Allow: p.Allow, Reason: p.Reason}) {
		// 未知 / 过期 id：可能客户端响应太晚，HTTP handler 已经超时
		h.logger.Warn("session", "permission.respond: unknown or expired id=%s", p.RequestID)
		return newErrorRespPtr(env.ID, protocol.ErrStaleRequest, "request not found or already resolved"), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func newErrorResp(id, code, msg string) Envelope {
	return Envelope{Type: "resp", ID: id, OK: boolPtr(false), Error: &RPCError{Code: code, Message: msg}}
}
func newErrorRespPtr(id, code, msg string) *Envelope {
	e := newErrorResp(id, code, msg)
	return &e
}
func boolPtr(b bool) *bool { return &b }
