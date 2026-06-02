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

// forward 把 runner 的事件复制到 manager 的 output，同时监控沉默。
func (m *Manager) forward(run engine.Runner) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	lastActivity := time.Now()

	for {
		select {
		case ev, ok := <-run.Events():
			if !ok {
				// Channel closed, runner exited
				m.mu.Lock()
				if m.active == run {
					m.active = nil
				}
				m.mu.Unlock()
				return
			}
			lastActivity = time.Now()
			m.out <- ev
		case <-ticker.C:
			if time.Since(lastActivity) > 120*time.Second {
				// Stall watchdog: kill runner
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
	return run.Write(p)
}

// SessionID 返回当前活跃 session id，无活跃返回空。
func (m *Manager) SessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sid
}
