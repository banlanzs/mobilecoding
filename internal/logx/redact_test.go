package logx

import "testing"

func TestRedactAuthorizationBearer(t *testing.T) {
	in := "Authorization: Bearer abc.def.ghi"
	out := Redact(in)
	want := "Authorization: Bearer <redacted>"
	if out != want {
		t.Errorf("Redact() = %q, want %q", out, want)
	}
}

func TestRedactTokenKeyValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"api_key=sk-live-12345", "api_key=<redacted>"},
		{"token: my-secret-token", "token=<redacted>"},
		{"PASSWORD=hunter2", "PASSWORD=<redacted>"},
		{"auth_token: t=abc", "auth_token=<redacted>"},
	}
	for _, c := range cases {
		got := Redact(c.in)
		if got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRedactNoMatch(t *testing.T) {
	in := "session started, no secrets here"
	out := Redact(in)
	if out != in {
		t.Errorf("Redact() should leave non-secret text untouched, got %q", out)
	}
}

func TestRedactEmpty(t *testing.T) {
	if got := Redact(""); got != "" {
		t.Errorf("Redact(\"\") = %q, want empty", got)
	}
}
