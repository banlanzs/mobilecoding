package hook

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRegistryRegisterRespond(t *testing.T) {
	reg := NewRegistry()
	if reg.Pending() != 0 {
		t.Fatalf("expected 0 pending, got %d", reg.Pending())
	}

	id, ch := reg.Register()
	if reg.Pending() != 1 {
		t.Fatalf("expected 1 pending, got %d", reg.Pending())
	}
	if !strings.HasPrefix(id, "permreq_") {
		t.Errorf("request id should start with permreq_, got %q", id)
	}

	if reg.Respond("nonexistent", Decision{Allow: true}) {
		t.Error("Respond on unknown id should return false")
	}

	if !reg.Respond(id, Decision{Allow: true, Reason: "ok"}) {
		t.Fatal("Respond on known id should return true")
	}
	if reg.Pending() != 0 {
		t.Errorf("after respond, expected 0 pending, got %d", reg.Pending())
	}

	select {
	case d := <-ch:
		if !d.Allow || d.Reason != "ok" {
			t.Errorf("unexpected decision: %+v", d)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel should have received a decision")
	}
}

func TestRegistryAbort(t *testing.T) {
	reg := NewRegistry()
	id, ch := reg.Register()
	reg.Abort(id)
	if reg.Pending() != 0 {
		t.Errorf("expected 0 pending after abort, got %d", reg.Pending())
	}
	_, ok := <-ch
	if ok {
		t.Error("channel should be closed after Abort")
	}
}

func TestHandlerServeHTTPAllow(t *testing.T) {
	reg := NewRegistry()
	var capturedEvent Event
	var mu sync.Mutex
	broadcast := func(ev Event) {
		mu.Lock()
		capturedEvent = ev
		mu.Unlock()
	}
	h := NewHandler(reg, broadcast)
	h.Timeout = 2 * time.Second

	body := `{
		"session_id": "test-session",
		"cwd": "/tmp",
		"hook_event_name": "PermissionRequest",
		"tool_name": "Bash",
		"tool_input": {"command": "ls -la", "description": "List files"}
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/hooks/permission-request", strings.NewReader(body))
	rr := httptest.NewRecorder()

	go func() {
		time.Sleep(50 * time.Millisecond)
		reg.mu.Lock()
		var id string
		for k := range reg.pending {
			id = k
			break
		}
		reg.mu.Unlock()
		if id == "" {
			t.Error("expected at least one pending request")
			return
		}
		reg.Respond(id, Decision{Allow: true, Reason: "looks safe"})
	}()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.HookSpecificOutput.HookEventName != "PermissionRequest" {
		t.Errorf("expected hook event name 'PermissionRequest', got %q", resp.HookSpecificOutput.HookEventName)
	}
	if resp.HookSpecificOutput.Decision.Behavior != "allow" {
		t.Errorf("expected behavior 'allow', got %q", resp.HookSpecificOutput.Decision.Behavior)
	}

	mu.Lock()
	defer mu.Unlock()
	if capturedEvent.Type != "permission_request" {
		t.Errorf("expected event type 'permission_request', got %q", capturedEvent.Type)
	}
	if capturedEvent.ToolName != "Bash" {
		t.Errorf("expected tool 'Bash', got %q", capturedEvent.ToolName)
	}
	if !strings.Contains(capturedEvent.ToolInputPrompt, "ls -la") {
		t.Errorf("expected prompt to contain command, got %q", capturedEvent.ToolInputPrompt)
	}
}

func TestHandlerServeHTTPDeny(t *testing.T) {
	reg := NewRegistry()
	h := NewHandler(reg, nil)
	h.Timeout = 2 * time.Second

	body := `{"tool_name": "Bash", "tool_input": {"command": "rm -rf /"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/hooks/permission-request", strings.NewReader(body))
	rr := httptest.NewRecorder()

	go func() {
		time.Sleep(50 * time.Millisecond)
		reg.mu.Lock()
		var id string
		for k := range reg.pending {
			id = k
		}
		reg.mu.Unlock()
		reg.Respond(id, Decision{Allow: false, Reason: "too dangerous"})
	}()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp Response
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.HookSpecificOutput.Decision.Behavior != "deny" {
		t.Errorf("expected deny, got %q", resp.HookSpecificOutput.Decision.Behavior)
	}
	if !strings.Contains(resp.HookSpecificOutput.Decision.Message, "dangerous") {
		t.Errorf("expected message to include reason, got %q", resp.HookSpecificOutput.Decision.Message)
	}
}

func TestHandlerServeHTTPTimeoutDeny(t *testing.T) {
	reg := NewRegistry()
	h := NewHandler(reg, nil)
	h.Timeout = 100 * time.Millisecond
	h.DenyOnTimeout = true

	body := `{"tool_name": "Bash", "tool_input": {"command": "sleep 999"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/hooks/permission-request", strings.NewReader(body))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp Response
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.HookSpecificOutput.Decision.Behavior != "deny" {
		t.Errorf("expected deny on timeout, got %q", resp.HookSpecificOutput.Decision.Behavior)
	}
	if !strings.Contains(resp.HookSpecificOutput.Decision.Message, "timeout") {
		t.Errorf("expected message to mention timeout, got %q", resp.HookSpecificOutput.Decision.Message)
	}
}

func TestHandlerRejectsNonPOST(t *testing.T) {
	h := NewHandler(NewRegistry(), nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/hooks/permission-request", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandlerRejectsMissingToolName(t *testing.T) {
	h := NewHandler(NewRegistry(), nil)
	body := `{"tool_name": ""}`
	req := httptest.NewRequest(http.MethodPost, "/v1/hooks/permission-request", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestSummarizeToolInput(t *testing.T) {
	cases := []struct {
		tool   string
		body   string
		prefix string
	}{
		{"Bash", `{"command": "ls -la", "description": "list"}`, "Bash: list"},
		{"Bash", `{"command": "ls -la"}`, "Bash: $ ls -la"},
		{"Write", `{"file_path": "/etc/hosts"}`, "Write: /etc/hosts"},
		{"Edit", `{"file_path": "/tmp/x.go"}`, "Edit: /tmp/x.go"},
		{"Read", `{"file_path": "/tmp/x.go"}`, "Read: /tmp/x.go"},
		{"Unknown", `{"foo": "bar"}`, "Unknown:"},
	}
	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			got := summarizeToolInput(tc.tool, json.RawMessage(tc.body))
			if !strings.HasPrefix(got, tc.prefix) {
				t.Errorf("summarizeToolInput(%q, %q) = %q; want prefix %q", tc.tool, tc.body, got, tc.prefix)
			}
		})
	}
}

func TestSettingsInjectorInstallAndUninstall(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.json"
	inj := NewSettingsInjector(path)

	cfg := HookConfig{
		URL:   "http://127.0.0.1:8443/v1/hooks/permission-request",
		Token: "test-token",
	}
	if err := inj.Install(cfg); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !inj.IsInstalled() {
		t.Error("IsInstalled should be true after Install")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	for _, want := range []string{"PermissionRequest", "_mobilecoding", "127.0.0.1:8443", "test-token"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("settings should contain %q, got: %s", want, data)
		}
	}

	if err := inj.Install(cfg); err != nil {
		t.Fatalf("second install: %v", err)
	}
	var settings map[string]any
	_ = json.Unmarshal(data, &settings)
	hooks, _ := settings["hooks"].(map[string]any)
	prList, _ := hooks["PermissionRequest"].([]any)
	if len(prList) != 1 {
		t.Errorf("expected 1 entry after duplicate install, got %d", len(prList))
	}

	if err := inj.Uninstall(); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if inj.IsInstalled() {
		t.Error("IsInstalled should be false after Uninstall")
	}
}

func TestSettingsInjectorPreservesExistingSettings(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.json"

	original := `{
  "theme": "dark",
  "model": "claude-sonnet-4-6",
  "permissions": {"allow": ["Read"]}
}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	inj := NewSettingsInjector(path)
	if err := inj.Install(HookConfig{URL: "http://x", Token: "y"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	for _, want := range []string{`"theme": "dark"`, `"model": "claude-sonnet-4-6"`, `"allow"`, `"Read"`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("settings should preserve %q, got: %s", want, data)
		}
	}
}

func TestSettingsInjectorBackupAndRestore(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/settings.json"
	original := `{"theme": "light"}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	inj := NewSettingsInjector(path)
	if err := inj.Install(HookConfig{URL: "http://x", Token: "y"}); err != nil {
		t.Fatal(err)
	}
	if err := inj.Uninstall(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"theme": "light"`) {
		t.Errorf("expected original content after uninstall, got: %s", data)
	}
	if strings.Contains(string(data), "_mobilecoding") {
		t.Errorf("uninstalled settings should not contain marker, got: %s", data)
	}
}
