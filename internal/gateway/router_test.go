package gateway

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http/httptest"
	"os"
	"testing"
)

//go:embed testdata/*
var testdata embed.FS

func newTestSPA() fs.FS {
	sub, err := fs.Sub(testdata, "testdata")
	if err != nil {
		panic(err)
	}
	return sub
}

func TestHealthz(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if rr.Body.String() != "ok" {
		t.Errorf("body = %q, want ok", rr.Body.String())
	}
}

func TestVersion(t *testing.T) {
	h := NewRouter(Dependencies{
		FS:          newTestSPA(),
		Version:     "0.1.0",
		DefaultCmd:  "claude",
		DefaultArgs: []string{"--model", "sonnet"},
		LaunchMode:  "remote-control",
	}, "test-token")
	req := httptest.NewRequest("GET", "/version", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["version"] != "0.1.0" {
		t.Errorf("version = %q, want 0.1.0", got["version"])
	}
	if got["runtime"] == nil {
		t.Errorf("runtime should be present in /version response")
	}
	runtime, ok := got["runtime"].(map[string]any)
	if !ok {
		t.Fatalf("runtime should be an object, got %T", got["runtime"])
	}
	if runtime["launchMode"] != "remote-control" {
		t.Errorf("runtime.launchMode = %q, want remote-control", runtime["launchMode"])
	}
}

func TestSPAFallback(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/some/unknown/route", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200 (SPA fallback)", rr.Code)
	}
	body := rr.Body.String()
	if body == "" {
		t.Errorf("body should not be empty")
	}
}

func TestWSEndpointAvailable(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	// WS endpoint is unauthenticated; returns 503 when WS handler is nil
	if rr.Code != 503 {
		t.Errorf("status = %d, want 503", rr.Code)
	}
}

// TestReadSettingsModelsUsesRealModelNameAsLabel 回归测试：
// label 必须是实际模型名（env 变量的值），而非固定的 Haiku/Sonnet/Opus 档位名。
func TestReadSettingsModelsUsesRealModelNameAsLabel(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.local.json"
	content := `{"env":{
		"ANTHROPIC_DEFAULT_HAIKU_MODEL": "minimax-m3[1m]",
		"ANTHROPIC_DEFAULT_OPUS_MODEL": "kimi-k2.7-code",
		"ANTHROPIC_DEFAULT_SONNET_MODEL": "deepseek-v4-pro[1m]",
		"ANTHROPIC_MODEL": "glm-5.2[1m]"
	}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	raw := readSettingsModels(path)
	models := parseModels(raw)

	// 第一项是"默认模型"（空 value）
	if models[0]["label"] != "默认模型" || models[0]["value"] != "" {
		t.Errorf("first entry = %v, want 默认模型:(empty)", models[0])
	}

	// 其余 label 必须是实际模型名，不能出现固定档位名 Haiku/Sonnet/Opus
	labels := map[string]bool{}
	for _, m := range models[1:] {
		labels[m["label"]] = true
		if m["label"] != m["value"] {
			t.Errorf("label %q != value %q, label should be the real model name", m["label"], m["value"])
		}
	}
	for _, fixed := range []string{"Haiku", "Sonnet", "Opus", "默认"} {
		if labels[fixed] {
			t.Errorf("fixed tier label %q should not appear, got real model names instead", fixed)
		}
	}
	for _, want := range []string{"minimax-m3[1m]", "kimi-k2.7-code", "deepseek-v4-pro[1m]", "glm-5.2[1m]"} {
		if !labels[want] {
			t.Errorf("expected model %q in list, missing", want)
		}
	}
}

func TestReadSettingsModelsPrefersExplicitModelsField(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.json"
	content := `{"models":"Sonnet:claude-sonnet-4-6,Opus:claude-opus-4-8","env":{"ANTHROPIC_MODEL":"ignored"}}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	raw := readSettingsModels(path)
	models := parseModels(raw)
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0]["label"] != "Sonnet" || models[0]["value"] != "claude-sonnet-4-6" {
		t.Errorf("models[0] = %v, want Sonnet:claude-sonnet-4-6", models[0])
	}
}

func TestReadSettingsModelsEmptyWhenNoModelEnv(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.json"
	if err := os.WriteFile(path, []byte(`{"env":{"OTHER_VAR":"x"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readSettingsModels(path); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
