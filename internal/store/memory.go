package store

// MemoryEntry 表示一条 memory。
type MemoryEntry struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// ListMemory 列出 memory。
func ListMemory(storeDir string) ([]MemoryEntry, error) {
	// MVP 2：简单占位
	return nil, nil
}
