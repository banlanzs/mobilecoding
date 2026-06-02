package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoadToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth", "token")

	tok, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken failed: %v", err)
	}
	if err := SaveToken(path, tok); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// 验证文件权限
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if runtime.GOOS != "windows" {
		if perm := st.Mode().Perm(); perm != 0o600 {
			t.Errorf("token file perm = %o, want 0o600", perm)
		}
	}

	got, err := LoadToken(path)
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if got != tok {
		t.Errorf("LoadToken = %q, want %q", got, tok)
	}
}

func TestLoadTokenMissing(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadToken(filepath.Join(dir, "nope")); err == nil {
		t.Errorf("LoadToken on missing file should fail")
	}
}

func TestSaveTokenRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := SaveToken(filepath.Join(dir, "tok"), ""); err == nil {
		t.Errorf("SaveToken with empty token should fail")
	}
}

func TestNewTokenIsRandom(t *testing.T) {
	a, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	b, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if a == b {
		t.Errorf("two NewToken() calls should produce different tokens")
	}
}
