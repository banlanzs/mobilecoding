// Package files 提供文件系统和 git 操作。
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileContent 是读文件的返回结构。
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Size     int64  `json:"size"`
	TooLarge bool   `json:"tooLarge"` // 超过 maxSize 时为 true，content 截断
	MaxSize  int64  `json:"maxSize"`
}

// ReadFile 读取 root 下 relPath 指定的文件内容。
// 安全性：解析后的绝对路径必须在 root 下，防止路径穿越（../）。
// maxSize 限制读取字节数，超出则截断并标记 TooLarge。
func ReadFile(root, relPath string, maxSize int) (*FileContent, error) {
	if root == "" {
		root = "."
	}
	if maxSize <= 0 {
		maxSize = 200 * 1024 // 默认 200KB
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	// 清洗 relPath，防止穿越
	cleanRel := filepath.Clean(relPath)
	if strings.HasPrefix(cleanRel, "..") || filepath.IsAbs(cleanRel) {
		return nil, fmt.Errorf("path outside workspace: %s", relPath)
	}
	absPath := filepath.Join(absRoot, cleanRel)

	// 再次确认解析后的路径在 root 下
	if !isWithin(absPath, absRoot) {
		return nil, fmt.Errorf("path outside workspace: %s", relPath)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("is a directory: %s", relPath)
	}

	fc := &FileContent{
		Path:    filepath.ToSlash(cleanRel),
		Size:    info.Size(),
		MaxSize: int64(maxSize),
	}

	if info.Size() > int64(maxSize) {
		fc.TooLarge = true
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		fc.Content = string(data[:maxSize])
	} else {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("read: %w", err)
		}
		fc.Content = string(data)
	}
	return fc, nil
}

// isWithin 检查 path 是否在 base 目录下（含 base 自身）。
func isWithin(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}
