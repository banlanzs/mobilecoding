package gateway

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestMemoryHandler_List(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA(), StoreDir: t.TempDir()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/memory", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestMemoryHandler_Update(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA(), StoreDir: t.TempDir()}, "test-token")
	body := `{"content":"new content"}`
	req := httptest.NewRequest("PUT", "/api/v1/memory/test", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}
