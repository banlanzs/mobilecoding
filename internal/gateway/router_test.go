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
	h := NewRouter(Dependencies{FS: newTestSPA(), Version: "0.1.0"}, "test-token")
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

func TestWSRejectsMissingToken(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 401 {
		t.Errorf("status = %d, want 401", rr.Code)
	}
}
