package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectSettingsPathUsesProjectLocalSettings(t *testing.T) {
	cwd := filepath.Join("tmp", "mobilecoding")
	want := filepath.Join(cwd, ".claude", "settings.local.json")
	if got := ProjectSettingsPath(cwd); got != want {
		t.Fatalf("ProjectSettingsPath() = %q, want %q", got, want)
	}
}

func TestRemoveInstalledHookOnlyRemovesMobilecodingMarker(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, "settings.json")
	backupPath := settingsPath + ".mobilecoding.bak"
	settings := map[string]any{
		"env": map[string]any{"KEEP": "1"},
		"hooks": map[string]any{
			"PermissionRequest": []any{
				map[string]any{
					"matcher": "Bash",
					"hooks":   []any{map[string]any{"type": "http", "url": "http://other"}},
				},
				map[string]any{
					"matcher": "",
					"hooks":   []any{map[string]any{"type": "http", "url": "http://mobilecoding", "_mobilecoding": "mobilecoding-hook"}},
				},
			},
		},
	}
	data, _ := json.Marshal(settings)
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte(`{"env":{"OLD":"backup"}}`), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	inj := NewSettingsInjector(settingsPath)
	if err := inj.RemoveInstalledHook(); err != nil {
		t.Fatalf("RemoveInstalledHook: %v", err)
	}

	updatedData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read updated settings: %v", err)
	}
	var updated map[string]any
	if err := json.Unmarshal(updatedData, &updated); err != nil {
		t.Fatalf("decode updated settings: %v", err)
	}
	if updated["env"].(map[string]any)["KEEP"] != "1" {
		t.Fatalf("existing settings should be preserved, got %v", updated["env"])
	}
	hooks := updated["hooks"].(map[string]any)["PermissionRequest"].([]any)
	if len(hooks) != 1 {
		t.Fatalf("PermissionRequest hooks len = %d, want 1", len(hooks))
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("RemoveInstalledHook should not restore/remove backup: %v", err)
	}
}
