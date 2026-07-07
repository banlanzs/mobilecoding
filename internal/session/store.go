// Package session 提供会话元数据持久化。
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SessionMeta 会话元数据，用于会话列表展示和恢复。
type SessionMeta struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`           // 用户自定义名称或自动生成
	Agent          string    `json:"agent"`          // claude/codex/opencode
	Model          string    `json:"model"`          // 使用的模型
	CWD            string    `json:"cwd"`            // 工作目录
	Status         string    `json:"status"`         // active/inactive/archived
	ResumeSessionID string   `json:"resumeSessionId,omitempty"` // Claude 内部 session_id，用于 --resume 续聊
	Command        string    `json:"command,omitempty"`         // 启动命令，用于恢复时重建 args
	Args           []string  `json:"args,omitempty"`            // 启动参数，用于恢复时重建
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	LastActiveAt   time.Time `json:"lastActiveAt"` // 最后活跃时间
	MessageCount   int       `json:"messageCount"` // 消息数量
}

// Store 会话元数据持久化存储（基于 JSON 文件）。
type Store struct {
	mu       sync.RWMutex
	path     string
	sessions map[string]*SessionMeta
}

// NewStore 创建或打开会话存储。
// path 为空时默认 ~/.mobilecoding/sessions.json。
func NewStore(path string) (*Store, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(home, ".mobilecoding", "sessions.json")
	}

	// 确保目录存在
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir: %w", err)
		}
	}

	s := &Store{
		path:     path,
		sessions: make(map[string]*SessionMeta),
	}

	// 加载现有数据
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

// load 从磁盘加载会话列表。
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var list []*SessionMeta
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("unmarshal sessions: %w", err)
	}

	s.sessions = make(map[string]*SessionMeta, len(list))
	for _, meta := range list {
		s.sessions[meta.ID] = meta
	}
	return nil
}

// save 持久化会话列表到磁盘。
func (s *Store) save() error {
	list := make([]*SessionMeta, 0, len(s.sessions))
	for _, meta := range s.sessions {
		list = append(list, meta)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}

	// 写入临时文件，然后原子重命名
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// Create 创建新会话元数据。
func (s *Store) Create(meta *SessionMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[meta.ID]; exists {
		return fmt.Errorf("session %s already exists", meta.ID)
	}

	now := time.Now()
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = now
	}
	if meta.UpdatedAt.IsZero() {
		meta.UpdatedAt = now
	}
	if meta.LastActiveAt.IsZero() {
		meta.LastActiveAt = now
	}
	if meta.Status == "" {
		meta.Status = "active"
	}

	s.sessions[meta.ID] = meta
	return s.save()
}

// Update 更新会话元数据。
func (s *Store) Update(id string, fn func(*SessionMeta)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, exists := s.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	fn(meta)
	meta.UpdatedAt = time.Now()
	return s.save()
}

// Get 获取单个会话元数据。
func (s *Store) Get(id string) (*SessionMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	meta, exists := s.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}

	// 返回副本，避免外部修改
	copied := *meta
	return &copied, nil
}

// List 返回所有会话元数据列表（按最后活跃时间降序）。
func (s *Store) List() []*SessionMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*SessionMeta, 0, len(s.sessions))
	for _, meta := range s.sessions {
		copied := *meta
		list = append(list, &copied)
	}

	// 按最后活跃时间降序排序
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[i].LastActiveAt.Before(list[j].LastActiveAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	return list
}

// Delete 删除会话元数据。
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[id]; !exists {
		return fmt.Errorf("session %s not found", id)
	}

	delete(s.sessions, id)
	return s.save()
}

// UpdateActivity 更新会话的最后活跃时间和消息计数。
func (s *Store) UpdateActivity(id string) error {
	return s.Update(id, func(meta *SessionMeta) {
		meta.LastActiveAt = time.Now()
		meta.MessageCount++
	})
}
