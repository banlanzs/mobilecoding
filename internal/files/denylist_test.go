package files

import (
	"testing"
)

func TestDenyListMatches(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{".env", true},
		{"path/to/.env", true},
		{"server.key", true},
		{"cert.pem", true},
		{"cert.p12", true},
		{"ca.crt", true},
		{".git/config", true},
		{"node_modules/foo", true},
		{"src/main.go", false},
		{"README.md", false},
	}
	for _, c := range cases {
		got := IsDenied(c.path)
		if got != c.want {
			t.Errorf("IsDenied(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}
