// Package files 提供文件系统和 git 操作。
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode 表示目录树中的一个节点。
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`     // 相对根目录的路径
	IsDir    bool       `json:"isDir"`
	Size     int64      `json:"size"`     // 文件字节数（目录为 0）
	Children []TreeNode `json:"children"` // 仅目录有
}

// ListTree 列出 root 目录下的文件树，最多展开到 maxDepth 层。
// 忽略 .git / node_modules 等目录，单目录最多列 maxEntries 个条目。
// root 必须存在且是目录，否则返回错误。
func ListTree(root string, maxDepth, maxEntries int) ([]TreeNode, error) {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", root)
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxEntries <= 0 {
		maxEntries = 200
	}
	return listDir(abs, "", 1, maxDepth, maxEntries)
}

var ignoreDirs = map[string]bool{
	".git": true, "node_modules": true, ".idea": true, ".vscode": true,
	"dist": true, "build": true, ".next": true, "__pycache__": true,
	".cache": true, "vendor": true, "Pods": true, ".gradle": true,
}

func listDir(absRoot, relPrefix string, depth, maxDepth, maxEntries int) ([]TreeNode, error) {
	if depth > maxDepth {
		return nil, nil
	}
	entries, err := os.ReadDir(absRoot)
	if err != nil {
		return nil, err
	}

	// 过滤 + 排序（目录优先，再按名字）
	var filtered []os.DirEntry
	for _, e := range entries {
		if e.IsDir() && ignoreDirs[e.Name()] {
			continue
		}
		// 跳过隐藏文件/目录（以 . 开头）
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		filtered = append(filtered, e)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].IsDir() != filtered[j].IsDir() {
			return filtered[i].IsDir()
		}
		return filtered[i].Name() < filtered[j].Name()
	})
	if len(filtered) > maxEntries {
		filtered = filtered[:maxEntries]
	}

	nodes := make([]TreeNode, 0, len(filtered))
	for _, e := range filtered {
		rel := filepath.Join(relPrefix, e.Name())
		node := TreeNode{
			Name:  e.Name(),
			Path:  filepath.ToSlash(rel),
			IsDir: e.IsDir(),
		}
		if !e.IsDir() {
			if info, err := e.Info(); err == nil {
				node.Size = info.Size()
			}
		} else {
			children, err := listDir(filepath.Join(absRoot, e.Name()), rel, depth+1, maxDepth, maxEntries)
			if err == nil {
				node.Children = children
			}
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
