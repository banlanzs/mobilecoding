package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	type Item struct {
		Name string `json:"name"`
	}
	item := Item{Name: "hello"}

	if err := SaveJSON(path, item); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}

	var loaded Item
	if err := LoadJSON(path, &loaded); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if loaded.Name != "hello" {
		t.Errorf("Name = %q, want hello", loaded.Name)
	}
}

func TestFileStoreAtomicRename(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")

	if err := SaveJSON(path, map[string]string{"a": "1"}); err != nil {
		t.Fatalf("SaveJSON: %v", err)
	}
	if err := SaveJSON(path, map[string]string{"a": "2"}); err != nil {
		t.Fatalf("SaveJSON overwrite: %v", err)
	}
	raw, _ := os.ReadFile(path)
	var m map[string]string
	json.Unmarshal(raw, &m)
	if m["a"] != "2" {
		t.Errorf("a = %q, want 2", m["a"])
	}
}

func TestLoadJSONMissing(t *testing.T) {
	dir := t.TempDir()
	var m map[string]string
	if err := LoadJSON(filepath.Join(dir, "nope.json"), &m); err == nil {
		t.Error("LoadJSON on missing file should fail")
	}
}
