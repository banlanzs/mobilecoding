package ws

import (
	"context"
	"encoding/json"

	"github.com/jaycrl/mytool/internal/engine"
	"github.com/jaycrl/mytool/internal/projection"
	"github.com/jaycrl/mytool/internal/session"
)

type Handler struct {
	hub *Hub
	mgr *session.Manager
}

func NewHandler(hub *Hub, mgr *session.Manager) *Handler {
	return &Handler{hub: hub, mgr: mgr}
}

func (h *Handler) ServeConn(ctx context.Context, c *Conn) error {
	sub := h.hub.Subscribe()
	defer h.hub.Unsubscribe(sub)

	go h.forwardSession(ctx, sub)

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
		_ = evt
	}
}

func (h *Handler) forwardSession(ctx context.Context, sub chan Envelope) {
	projOut := make(chan projection.Event, 64)
	go func() {
		for ev := range projOut {
			env, err := projectionToEnvelope(ev)
			if err != nil {
				continue
			}
			select {
			case sub <- env:
			default:
				// 背压：丢弃
			}
		}
	}()
	sid := h.mgr.SessionID()
	projection.Stream(h.mgr.Output(), projOut, sid)
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

func newErrorResp(id, code, msg string) Envelope {
	return Envelope{Type: "resp", ID: id, OK: boolPtr(false), Error: &RPCError{Code: code, Message: msg}}
}
func newErrorRespPtr(id, code, msg string) *Envelope {
	e := newErrorResp(id, code, msg)
	return &e
}
func boolPtr(b bool) *bool { return &b }
