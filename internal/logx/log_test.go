package logx

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestLoggerRedactsMessage(t *testing.T) {
	var buf bytes.Buffer
	lg := NewWithWriter(&buf)
	lg.Info("auth", "user logged in with api_key=sk-live-12345")
	out := buf.String()
	if !strings.Contains(out, "api_key=<redacted>") {
		t.Errorf("logger should redact api_key in output, got: %s", out)
	}
	if strings.Contains(out, "sk-live-12345") {
		t.Errorf("logger should NOT contain raw secret, got: %s", out)
	}
}

func TestLoggerRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	lg := NewWithWriter(&buf)
	lg.SetLevel(LevelWarn)
	lg.Info("c", "info line")
	lg.Warn("c", "warn line")
	out := buf.String()
	if strings.Contains(out, "info line") {
		t.Errorf("info should be filtered out at warn level, got: %s", out)
	}
	if !strings.Contains(out, "warn line") {
		t.Errorf("warn line should be present, got: %s", out)
	}
}

func TestLoggerConcurrentSetLevel(t *testing.T) {
	lg := NewWithWriter(io.Discard)
	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ {
			lg.SetLevel(LevelInfo)
			lg.SetLevel(LevelWarn)
			lg.SetLevel(LevelError)
		}
		close(done)
	}()
	for i := 0; i < 1000; i++ {
		lg.Info("c", "iter=%d", i)
		lg.Warn("c", "iter=%d", i)
		lg.Error("c", "iter=%d", i)
	}
	<-done
}
