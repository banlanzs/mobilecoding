package main

import (
	"crypto/tls"
	"path/filepath"
	"testing"

	"github.com/jaycrl/mytool/internal/auth"
)

func TestTLSConfig_OptionalMode(t *testing.T) {
	dir := t.TempDir()
	ca, err := auth.LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("ca: %v", err)
	}
	if err := auth.LoadOrCreateServerCert(ca, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"), "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("server cert: %v", err)
	}

	tlsCfg, err := buildTLSConfig("optional", ca, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"))
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("expected non-nil config")
	}
	if tlsCfg.ClientAuth != tls.NoClientCert {
		t.Errorf("optional mode should not require client cert, got %v", tlsCfg.ClientAuth)
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Errorf("server cert should be loaded")
	}
}

func TestTLSConfig_RequiredMode(t *testing.T) {
	dir := t.TempDir()
	ca, _ := auth.LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	_ = auth.LoadOrCreateServerCert(ca, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"), "10.0.0.1", "box.lan")

	tlsCfg, err := buildTLSConfig("required", ca, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"))
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if tlsCfg.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Errorf("required mode should require client cert, got %v", tlsCfg.ClientAuth)
	}
}

func TestTLSConfig_RejectsNone(t *testing.T) {
	_, err := buildTLSConfig("none", nil, "", "")
	if err == nil {
		t.Errorf("none mode should be rejected in MVP 1.1")
	}
}
