// Package session 管理单个活跃 runner 的生命周期与事件转发。
// MVP 1 阶段：仅一个活跃 session；后续可扩展 resume / permission router。
package session

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/banlanzs/mobilecoding/internal/engine"
)

// Event 是 session 包暴露的最小事件类型（直接转 engine.Event）。
type Event = engine.Event

// ExecRequest 透传 engine.ExecRequest，避免上层 import engine。
type ExecRequest = engine.ExecRequest

// Manager 持有当前活跃 runner 与一条到订阅者的输出流。
// 支持会话元数据持久化和多会话管理。
type Manager struct {
	mu       sync.Mutex
	active   engine.Runner
	sid      string
	out      chan Event
	err      chan error
	once     sync.Once
	log      func(string, string, ...any) // component, format, args...
	store    *Store                       // 会话元数据存储（可选）
	metadata *SessionMeta                 // 当前活跃会话的元数据
}

// NewManager 构造一个空 manager。
func NewManager() *Manager {
	return &Manager{
		out: make(chan Event, 256),
		err: make(chan error, 8),
		log: func(string, string, ...any) {},
	}
}

// SetStore 注入会话存储（可选，用于持久化会话元数据）。
func (m *Manager) SetStore(store *Store) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = store
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

	// 创建会话元数据
	if m.store != nil {
		m.metadata = &SessionMeta{
			ID:              m.sid,
			Name:            generateSessionName(req.Command),
			Agent:           req.Command,
			Model:           extractModelFromArgs(req.Args),
			CWD:             req.CWD,
			Status:          "active",
			Command:         req.Command,
			Args:            req.Args,
			ResumeSessionID: req.ResumeSessionID,
		}
		if err := m.store.Create(m.metadata); err != nil {
			m.log("session", "failed to save session metadata: %v", err)
			// 不阻塞会话启动
		}
		// 注入 resume ID 回调：runner 捕获到新 session_id 时持久化，供跨进程恢复
		if setter, ok := engine.Runner(run).(engine.ResumeIDCallbackSetter); ok {
			sid := m.sid
			setter.SetResumeIDCallback(func(resumeID string) {
				if err := m.store.Update(sid, func(meta *SessionMeta) {
					meta.ResumeSessionID = resumeID
				}); err != nil {
					m.log("session", "failed to persist resume id: %v", err)
				}
			})
		}
	}
	m.mu.Unlock()

	if err := run.Start(ctx, req); err != nil {
		m.mu.Lock()
		m.active = nil
		m.sid = ""
		m.metadata = nil
		m.mu.Unlock()
		m.log("session", "runner start FAILED: command=%s err=%v", req.Command, err)
		return "", err
	}

	m.log("session", "runner started: command=%s sessionId=%s", req.Command, m.sid)
	go m.forward(run)
	return m.sid, nil
}

// Restart 替换当前 runner，但保留 mobilecoding 会话 ID 和历史归属。
func (m *Manager) Restart(ctx context.Context, req ExecRequest, run engine.Runner) (string, error) {
	m.mu.Lock()
	old := m.active
	if old == nil {
		m.mu.Unlock()
		return m.Start(ctx, req, run)
	}
	sid := m.sid
	m.active = nil
	m.mu.Unlock()

	if err := old.Close(); err != nil {
		return "", err
	}

	m.mu.Lock()
	m.active = run
	if m.metadata != nil {
		m.metadata.Agent = req.Command
		m.metadata.Model = extractModelFromArgs(req.Args)
		m.metadata.CWD = req.CWD
		m.metadata.Status = "active"
		m.metadata.Command = req.Command
		m.metadata.Args = req.Args
		m.metadata.ResumeSessionID = req.ResumeSessionID
	}
	if m.store != nil && sid != "" {
		if err := m.store.Update(sid, func(meta *SessionMeta) {
			meta.Agent = req.Command
			meta.Model = extractModelFromArgs(req.Args)
			meta.CWD = req.CWD
			meta.Status = "active"
			meta.Command = req.Command
			meta.Args = req.Args
			meta.ResumeSessionID = req.ResumeSessionID
		}); err != nil {
			m.log("session", "failed to update session metadata: %v", err)
		}
		// 注入 resume ID 回调（同 Start）
		if setter, ok := engine.Runner(run).(engine.ResumeIDCallbackSetter); ok {
			setter.SetResumeIDCallback(func(resumeID string) {
				if err := m.store.Update(sid, func(meta *SessionMeta) {
					meta.ResumeSessionID = resumeID
				}); err != nil {
					m.log("session", "failed to persist resume id: %v", err)
				}
			})
		}
	}
	m.mu.Unlock()

	if err := run.Start(ctx, req); err != nil {
		m.mu.Lock()
		if m.active == run {
			m.active = nil
		}
		m.mu.Unlock()
		m.log("session", "runner restart FAILED: command=%s err=%v", req.Command, err)
		return "", err
	}

	m.log("session", "runner restarted: command=%s sessionId=%s", req.Command, sid)
	go m.forward(run)
	return sid, nil
}

// generateSessionName 生成会话名称（基于 agent 和时间戳）。
func generateSessionName(agent string) string {
	return agent + "-" + uuid.NewString()[:8]
}

// extractModelFromArgs 从 args 中提取 --model 参数值。
func extractModelFromArgs(args []string) string {
	for i, arg := range args {
		if arg == "--model" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// forward 把 runner 的事件复制到 manager 的 output。
// 会话只在用户手动 Stop 或进程自然退出时结束，不会自动超时断开。
func (m *Manager) forward(run engine.Runner) {
	count := 0
	errCh := run.Errors()

	for {
		select {
		case ev, ok := <-run.Events():
			if !ok {
				m.mu.Lock()
				if m.active == run {
					m.active = nil
					// 更新会话状态为 inactive
					if m.store != nil && m.sid != "" {
						_ = m.store.Update(m.sid, func(meta *SessionMeta) {
							meta.Status = "inactive"
						})
					}
					m.sid = ""
					m.metadata = nil
				}
				m.mu.Unlock()
				m.log("session", "runner exited (events closed), forwarded %d events", count)
				return
			}
			select {
			case m.out <- ev:
				// 更新活跃时间
				m.mu.Lock()
				if m.store != nil && m.sid != "" {
					_ = m.store.UpdateActivity(m.sid)
				}
				m.mu.Unlock()
			default:
				m.log("session", "backpressure: out channel full, dropping event #%d kind=%s", count, ev.Kind)
			}
			count++
			if count <= 5 || count%50 == 0 {
				m.log("session", "event #%d kind=%s len=%d", count, ev.Kind, len(ev.Data))
			}
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				m.log("session", "runner stderr: %v", err)
			}
		}
	}
}

// Stop 关闭当前活跃 session。
func (m *Manager) Stop() error {
	m.mu.Lock()
	run := m.active
	sid := m.sid
	m.active = nil
	m.sid = ""
	m.metadata = nil
	m.mu.Unlock()

	if run == nil {
		return nil
	}

	// 更新会话状态为 inactive
	if m.store != nil && sid != "" {
		_ = m.store.Update(sid, func(meta *SessionMeta) {
			meta.Status = "inactive"
		})
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

// Abort 中止当前请求，保留 session 等待下一条消息。
func (m *Manager) Abort() {
	m.mu.Lock()
	run := m.active
	m.mu.Unlock()
	if run != nil {
		run.Abort()
	}
}

// SendToStdin 写入当前活跃 runner 的 stdin（不杀进程）。
func (m *Manager) SendToStdin(p []byte) error {
	m.mu.Lock()
	run := m.active
	m.mu.Unlock()
	if run == nil {
		return errors.New("session: no active runner")
	}
	return run.SendToStdin(p)
}

// SessionID 返回当前活跃 session id，无活跃返回空。
func (m *Manager) SessionID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sid
}

// ResumeSessionID 返回当前活跃 runner 的 Claude resume session ID。
// 用于跨会话恢复（Local/Remote 模式切换）。
func (m *Manager) ResumeSessionID() string {
	m.mu.Lock()
	run := m.active
	m.mu.Unlock()
	if run == nil {
		return ""
	}
	if p, ok := run.(engine.ResumeSessionIDProvider); ok {
		return p.GetResumeSessionID()
	}
	return ""
}
