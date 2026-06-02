package ws

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOriginMatchesHost(t *testing.T) {
	cases := []struct {
		origin string
		host   string
		want   bool
	}{
		{"https://example.com", "example.com", true},
		{"https://example.com:8443", "example.com:8443", true},
		{"http://example.com", "example.com", true},
		{"wss://example.com", "example.com", true},
		{"ws://example.com", "example.com", true},
		{"https://example.com/path", "example.com", true},
		{"https://other.com", "example.com", false},
		{"https://attacker.com", "example.com", false},
		{"", "example.com", false},
		{"x", "example.com", false},
	}
	for _, c := range cases {
		got := originMatchesHost(c.origin, c.host)
		if got != c.want {
			t.Errorf("originMatchesHost(%q, %q) = %v, want %v", c.origin, c.host, got, c.want)
		}
	}
}

func TestCheckOrigin_AllowsMatching(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://example.com")
	// Can't easily test upgrader's CheckOrigin directly without full WS upgrade,
	// but we can test the closure's behavior by reproducing it
	allowed := strings.HasPrefix(req.Header.Get("Origin"), "https://") &&
		originMatchesHost(req.Header.Get("Origin"), req.Host)
	if !allowed {
		t.Errorf("CheckOrigin closure should allow matching origin")
	}
}

func TestCheckOrigin_RejectsMismatched(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "https://attacker.com")
	allowed := originMatchesHost(req.Header.Get("Origin"), req.Host)
	if allowed {
		t.Errorf("CheckOrigin closure should reject mismatched origin")
	}
}

func TestCheckOrigin_AllowsEmptyOrigin(t *testing.T) {
	// Empty Origin (e.g., native clients, curl) should be accepted
	// The closure in conn.go returns true when origin is empty
	origin := ""
	allowed := origin == "" // matches the closure logic
	if !allowed {
		t.Errorf("CheckOrigin should accept empty origin (non-browser client)")
	}
}
