package files

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TreeNode 表示文件树的一个节点。
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"isDir"`
	Children []TreeNode `json:"children,omitempty"`
}

// ListTree 列出工作区文件树，限制深度，忽略 denylist。
func ListTree(root string, depth int) ([]TreeNode, error) {
	if depth <= 0 {
		depth = 3
	}
	return listDir(root, "", depth)
}

func listDir(root, rel string, depth int) ([]TreeNode, error) {
	if depth <= 0 {
		return nil, nil
	}
	abs := filepath.Join(root, rel)
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var nodes []TreeNode
	for _, e := range entries {
		childRel := filepath.Join(rel, e.Name())
		if IsDenied(childRel) {
			continue
		}
		node := TreeNode{
			Name:  e.Name(),
			Path:  childRel,
			IsDir: e.IsDir(),
		}
		if e.IsDir() {
			children, err := listDir(root, childRel, depth-1)
			if err != nil {
				continue
			}
			node.Children = children
		}
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}
