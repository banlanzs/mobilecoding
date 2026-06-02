package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListTree(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "lib.go"), []byte("package src"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(""), 0o644)

	nodes, err := ListTree(dir, 3)
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	// Should have main.go and src/, but not .git
	if len(nodes) != 2 {
		t.Errorf("len(nodes) = %d, want 2", len(nodes))
	}
}
