package files

import (
	"errors"
	"os"
	"path/filepath"
)

var allowedExtensions = map[string]bool{
	".go": true, ".js": true, ".ts": true, ".tsx": true,
	".py": true, ".java": true, ".rs": true, ".c": true,
	".cpp": true, ".h": true, ".md": true, ".txt": true,
	".json": true, ".yaml": true, ".yml": true, ".toml": true,
	".xml": true, ".html": true, ".css": true, ".sh": true,
	".bat": true, ".ps1": true, ".sql": true, ".graphql": true,
}

// ReadFile 读取工作区内的文件（白名单扩展名 + denylist 检查）。
func ReadFile(workspace, relPath string, maxBytes int) ([]byte, error) {
	if IsDenied(relPath) {
		return nil, errors.New("access denied")
	}
	ext := filepath.Ext(relPath)
	if !allowedExtensions[ext] {
		return nil, errors.New("file type not allowed")
	}
	abs := filepath.Join(workspace, relPath)
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if maxBytes > 0 && len(data) > maxBytes {
		data = data[:maxBytes]
	}
	return data, nil
}
