// Package session 管理单个活跃 runner 的生命周期与事件转发。
// MVP 1 阶段：仅一个活跃 session；后续可扩展 resume / permission router。
package session

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/banlanzs/mobilecoding/internal/engine"
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
	log    func(string, string, ...any) // component, format, args...
}

// NewManager 构造一个空 manager。
func NewManager() *Manager {
	return &Manager{
		out: make(chan Event, 64),
		err: make(chan error, 8),
		log: func(string, string, ...any) {},
	}
}

// SetLogger 注入日志函数（由 main 调用）。
func (m *Manager) SetLogger(log func(string, string, ...any)) {
	m.log = log
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
		m.log("session", "runner start FAILED: command=%s err=%v", req.Command, err)
		return "", err
	}

	m.log("session", "runner started: command=%s sessionId=%s", req.Command, m.sid)
	go m.forward(run)
	return m.sid, nil
}

// forward 把 runner 的事件复制到 manager 的 output，同时监控沉默。
func (m *Manager) forward(run engine.Runner) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	lastActivity := time.Now()
	count := 0
	errCh := run.Errors() // 闭合后设为 nil，避免 select 旋转

	for {
		select {
		case ev, ok := <-run.Events():
			if !ok {
				m.mu.Lock()
				if m.active == run {
					m.active = nil
				}
				m.mu.Unlock()
				m.log("session", "runner exited (events closed), forwarded %d events", count)
				return
			}
			lastActivity = time.Now()
			m.out <- ev
			count++
			if count <= 5 || count%50 == 0 {
				m.log("session", "event #%d kind=%s len=%d", count, ev.Kind, len(ev.Data))
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil // 闭合后屏蔽，防止 select 旋转
				continue
			}
			if err != nil {
				m.log("session", "runner stderr: %v", err)
			}
		case <-ticker.C:
			if time.Since(lastActivity) > 300*time.Second {
				m.log("session", "stall watchdog: killing runner")
				run.Close()
				m.mu.Lock()
				if m.active == run {
					m.active = nil
				}
				m.mu.Unlock()
				return
			}
		}
	}
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
	m.log("session", "stopping runner")
	return run.Close()
}

// Write 把 p 写入当前活跃 runner。
func (m *Manager) Write(p []byte) error {
	m.mu.Lock()
	run := m.active
	m.mu.Unlock()
	if run == nil {
		return errors.New("session: no active runner")
	}
	m.log("session", "write: %d bytes", len(p))
	return run.Write(p)
}

// SessionID 返回当前活跃 session id，无活跃返回空。
func (m *Manager) SessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sid
}
