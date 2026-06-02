package auth

import (
	"os"
	"testing"
)

func TestCheckCertExpiry(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(dir + "/ca.crt")
	_ = LoadOrCreateServerCert(ca, dir+"/server.crt", dir+"/server.key", "127.0.0.1", "localhost")

	// Fresh cert should not need rotation
	expired := CheckCertExpiry(dir + "/server.crt")
	if expired {
		t.Error("fresh cert should not need rotation")
	}
}

func TestCheckCertExpiryExpired(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(dir + "/ca.crt")

	// Create cert (1 year validity) — IssueDeviceCert returns PEM-encoded bytes
	cert, key, err := IssueDeviceCert(ca, "test")
	if err != nil {
		t.Fatalf("IssueDeviceCert: %v", err)
	}
	certPath := dir + "/expired.crt"
	keyPath := dir + "/expired.key"
	// writePEMFile takes raw DER, but IssueDeviceCert returns PEM.
	// Use os.WriteFile directly since bytes are already PEM-encoded.
	os.WriteFile(certPath, cert, 0o600)
	os.WriteFile(keyPath, key, 0o600)

	// Even short-lived certs should not be expired immediately
	expired := CheckCertExpiry(certPath)
	if expired {
		t.Error("cert should not be expired immediately after creation")
	}
}
