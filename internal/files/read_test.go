package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package test"), 0o644)

	data, err := ReadFile(dir, "test.go", 0)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "package test" {
		t.Errorf("data = %q, want package test", string(data))
	}
}

func TestReadFileDeniesEnv(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=abc"), 0o644)

	_, err := ReadFile(dir, ".env", 0)
	if err == nil {
		t.Error("ReadFile should deny .env")
	}
}

func TestReadFileDeniesKey(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "server.key"), []byte("key"), 0o644)

	_, err := ReadFile(dir, "server.key", 0)
	if err == nil {
		t.Error("ReadFile should deny .key")
	}
}
