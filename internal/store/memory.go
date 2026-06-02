package store

import (
	"os"
	"path/filepath"
)

// MemoryEntry 表示一条 memory。
type MemoryEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ListMemory 列出 memory。
func ListMemory(storeDir string) ([]MemoryEntry, error) {
	dir := filepath.Join(storeDir, "memory")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []MemoryEntry{}, nil
		}
		return nil, err
	}
	var result []MemoryEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		result = append(result, MemoryEntry{
			Name:    e.Name(),
			Content: string(data),
		})
	}
	return result, nil
}

// SaveMemory 保存一条 memory。
func SaveMemory(storeDir, name, content string) error {
	dir := filepath.Join(storeDir, "memory")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
}
