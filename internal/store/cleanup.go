package store

import "time"

// CleanupOldSessions 清理超过指定天数的旧消息。
// 返回删除的消息数。
func (s *MessageStore) CleanupOldSessions(keepDays int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(keepDays) * 24 * time.Hour).UTC().Format(time.RFC3339)
	result, err := s.db.Exec(
		`DELETE FROM session_messages WHERE created_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
