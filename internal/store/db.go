// store 包提供嵌入式 SQLite 消息持久化，支持序列号分页查询。
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// MessageStore 管理会话消息的持久化存储。
type MessageStore struct {
	db *sql.DB
}

// SequencedMessage 带序列号的消息记录。
type SequencedMessage struct {
	Seq       int64  `json:"seq"`
	SessionID string `json:"sessionId"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// Open 打开（或创建）SQLite 数据库并执行迁移。
// path 为空时默认 ~/.mobilecoding/messages.db。
func Open(path string) (*MessageStore, error) {
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".mobilecoding", "messages.db")
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir db dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &MessageStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS session_messages (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			seq        INTEGER NOT NULL,
			type       TEXT NOT NULL,
			content    TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_session_messages_seq
			ON session_messages(session_id, seq);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_session_messages_session_seq
			ON session_messages(session_id, seq);
	`)
	return err
}

// Close 关闭数据库连接。
func (s *MessageStore) Close() error {
	return s.db.Close()
}
