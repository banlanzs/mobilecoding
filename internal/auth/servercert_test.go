package auth

import (
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadOrCreateServerCert_New(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	ca, err := LoadOrCreateCA(caPath)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "192.168.1.10", "myhost.local"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}

	raw, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read server cert: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	if _, err := parseAndVerifyCert(raw, pool); err != nil {
		t.Errorf("server cert should be valid: %v", err)
	}

	st, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key: %v", err)
	}
	if runtime.GOOS != "windows" {
		if st.Mode().Perm() != 0o600 {
			t.Errorf("key perm = %o, want 0o600", st.Mode().Perm())
		}
	}
}

func TestServerCertSAN(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("LoadOrCreateServerCert: %v", err)
	}
	raw, _ := os.ReadFile(certPath)
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	cert, err := parseAndVerifyCert(raw, pool)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	hasIP, hasDNS := false, false
	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("10.0.0.1")) {
			hasIP = true
		}
	}
	for _, name := range cert.DNSNames {
		if name == "box.lan" {
			hasDNS = true
		}
	}
	if !hasIP {
		t.Errorf("SAN should include IP 10.0.0.1, got %v", cert.IPAddresses)
	}
	if !hasDNS {
		t.Errorf("SAN should include DNS box.lan, got %v", cert.DNSNames)
	}
}

func TestLoadOrCreateServerCert_Existing(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	certPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")

	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("first: %v", err)
	}
	stat1, _ := os.Stat(certPath)
	mod1 := stat1.ModTime()

	if err := LoadOrCreateServerCert(ca, certPath, keyPath, "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("second: %v", err)
	}
	stat2, _ := os.Stat(certPath)
	if !stat2.ModTime().Equal(mod1) {
		t.Errorf("second call should not regenerate server cert")
	}
}
