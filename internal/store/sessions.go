package store

import (
	"path/filepath"
)

// SessionRecord 表示一个会话记录。
type SessionRecord struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	StartedAt string `json:"startedAt"`
	StoppedAt string `json:"stoppedAt,omitempty"`
}

// SaveSession 保存会话记录。
func SaveSession(storeDir string, rec SessionRecord) error {
	path := filepath.Join(storeDir, "sessions", rec.ID+".json")
	return SaveJSON(path, rec)
}

// LoadSession 加载会话记录。
func LoadSession(storeDir, id string) (SessionRecord, error) {
	path := filepath.Join(storeDir, "sessions", id+".json")
	var rec SessionRecord
	if err := LoadJSON(path, &rec); err != nil {
		return rec, err
	}
	return rec, nil
}
