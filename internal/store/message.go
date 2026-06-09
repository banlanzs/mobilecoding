package store

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/banlanzs/mobilecoding/internal/projection"
)

// SaveMessage 保存投影事件到数据库，返回分配的 seq。
// 事件的 Seq 字段会被忽略，由 store 自动分配。
func (s *MessageStore) SaveMessage(sessionID string, ev projection.Event) (int64, error) {
	content, err := json.Marshal(ev)
	if err != nil {
		return 0, fmt.Errorf("marshal event: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 分配 seq：取当前 session 最大 seq + 1
	var seq int64
	err = tx.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM session_messages WHERE session_id = ?`,
		sessionID,
	).Scan(&seq)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("allocate seq: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO session_messages (id, session_id, seq, type, content) VALUES (?, ?, ?, ?, ?)`,
		ev.MessageID, sessionID, seq, ev.Type, string(content),
	)
	if err != nil {
		return 0, fmt.Errorf("insert message: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return seq, nil
}

// GetMessagesAfter 返回 seq > afterSeq 的消息（正向同步），最多 limit 条。
func (s *MessageStore) GetMessagesAfter(sessionID string, afterSeq int64, limit int) ([]SequencedMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	// 多取一条判断 hasMore
	rows, err := s.db.Query(
		`SELECT seq, session_id, type, content, created_at
		 FROM session_messages
		 WHERE session_id = ? AND seq > ?
		 ORDER BY seq ASC
		 LIMIT ?`,
		sessionID, afterSeq, limit+1,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	return scanMessages(rows, limit)
}

// GetMessagesBefore 返回 seq < beforeSeq 的消息（反向加载历史），最多 limit 条。
func (s *MessageStore) GetMessagesBefore(sessionID string, beforeSeq int64, limit int) ([]SequencedMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(
		`SELECT seq, session_id, type, content, created_at
		 FROM session_messages
		 WHERE session_id = ? AND seq < ?
		 ORDER BY seq DESC
		 LIMIT ?`,
		sessionID, beforeSeq, limit+1,
	)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	msgs, err := scanMessages(rows, limit)
	if err != nil {
		return nil, err
	}
	// 反转为正序
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func scanMessages(rows *sql.Rows, limit int) ([]SequencedMessage, error) {
	var msgs []SequencedMessage
	for rows.Next() {
		var m SequencedMessage
		if err := rows.Scan(&m.Seq, &m.SessionID, &m.Type, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return msgs, nil
}

// GetLatestSeq 返回指定 session 的最新 seq（无消息时返回 0）。
func (s *MessageStore) GetLatestSeq(sessionID string) (int64, error) {
	var seq int64
	err := s.db.QueryRow(
		`SELECT COALESCE(MAX(seq), 0) FROM session_messages WHERE session_id = ?`,
		sessionID,
	).Scan(&seq)
	return seq, err
}
