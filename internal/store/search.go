package store

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SearchResult 搜索结果。
type SearchResult struct {
	Seq       int64  `json:"seq"`
	SessionID string `json:"sessionId"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	Snippet   string `json:"snippet"` // 匹配片段
}

// SearchMessages 在消息内容中搜索关键词（模糊匹配）。
func (s *MessageStore) SearchMessages(sessionID, query string, limit int) ([]SearchResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if query == "" {
		return nil, nil
	}

	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.Query(
		`SELECT seq, session_id, type, content, created_at
		 FROM session_messages
		 WHERE session_id = ? AND LOWER(content) LIKE ?
		 ORDER BY seq DESC
		 LIMIT ?`,
		sessionID, pattern, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.Seq, &r.SessionID, &r.Type, &r.Content, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Snippet = extractSnippet(r.Content, query)
		results = append(results, r)
	}
	return results, rows.Err()
}

// extractSnippet 从 JSON 内容中提取包含查询词的文本片段。
func extractSnippet(content, query string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(content), &m); err != nil {
		return ""
	}
	text, _ := m["text"].(string)
	if text == "" {
		text, _ = m["message"].(string)
	}
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	idx := strings.Index(lower, strings.ToLower(query))
	if idx < 0 {
		if len(text) > 100 {
			return text[:100] + "..."
		}
		return text
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + len(query) + 40
	if end > len(text) {
		end = len(text)
	}
	snippet := text[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}
	return snippet
}
