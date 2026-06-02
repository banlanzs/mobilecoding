package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestSkillsHandler_List(t *testing.T) {
	h := NewRouter(Dependencies{FS: newTestSPA(), Workspace: t.TempDir()}, "test-token")
	req := httptest.NewRequest("GET", "/api/v1/skills", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got == nil {
		t.Error("expected non-nil array")
	}
}
