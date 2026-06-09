package gateway

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http/httptest"
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
