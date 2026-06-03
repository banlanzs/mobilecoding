package ws

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/banlanzs/mobilecoding/internal/engine"
	"github.com/banlanzs/mobilecoding/internal/logx"
	"github.com/banlanzs/mobilecoding/internal/projection"
	"github.com/banlanzs/mobilecoding/internal/session"
)

type Handler struct {
	hub    *Hub
	mgr    *session.Manager
	logger *logx.Logger
}

func NewHandler(hub *Hub, mgr *session.Manager, logger *logx.Logger) *Handler {
	return &Handler{hub: hub, mgr: mgr, logger: logger}
}

func (h *Handler) ServeConn(ctx context.Context, c *Conn) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	writeCh := make(chan Envelope, 128)

	// 桥接协程：writeCh → c.send（writeLoop 独占 WebSocket）
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case env, ok := <-writeCh:
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

	go h.forwardSession(ctx, writeCh)

	// 阻塞发送辅助函数
	sendResp := func(env Envelope) {
		select {
		case writeCh <- env:
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
			sendResp(newErrorResp(env.ID, "protocol_error", "unsupported envelope type"))
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

	h.logger.Info("session", "starting: command=%s args=%v cwd=%s", p.Command, p.Args, p.CWD)

	run, err := engine.NewRunner(p.Command, engine.ExecRequest{
		Command: p.Command,
		Args:    p.Args,
		CWD:     p.CWD,
	})
	if err != nil {
		h.logger.Error("session", "new runner failed: command=%s err=%v", p.Command, err)
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	sid, err := h.mgr.Start(ctx, engine.ExecRequest{Command: p.Command, Args: p.Args, CWD: p.CWD}, run)
	if err != nil {
		h.logger.Error("session", "start failed: command=%s err=%v", p.Command, err)
		return newErrorRespPtr(env.ID, "conflict", err.Error()), nil
	}
	result, _ := json.Marshal(map[string]string{"sessionId": sid})
	ok := true
	h.logger.Info("session", "started: command=%s sessionId=%s", p.Command, sid)
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
	inp := strings.TrimSpace(p.Text)
	if len(inp) > 50 {
		inp = inp[:50] + "..."
	}
	h.logger.Debug("session", "input: sessionId=%s text=%q", p.SessionID, inp)
	if err := h.mgr.Write([]byte(p.Text + "\n")); err != nil {
		h.logger.Error("session", "write input failed: err=%v", err)
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	ok := true
	return &Envelope{Type: "resp", ID: env.ID, OK: &ok}, nil
}

func (h *Handler) handleStop(env Envelope) (*Envelope, any) {
	if err := h.mgr.Stop(); err != nil {
		return newErrorRespPtr(env.ID, "engine_failure", err.Error()), nil
	}
	h.logger.Info("session", "stopped")
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