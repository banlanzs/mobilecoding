package auth

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOrCreateCA_New(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	ca, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	if ca.Certificate == nil {
		t.Fatal("CA cert should not be nil")
	}
	if !ca.Certificate.IsCA {
		t.Error("cert should be a CA")
	}
	if !ca.Certificate.NotAfter.After(time.Now().Add(365 * 24 * time.Hour)) {
		t.Error("CA should be valid for at least 1 year")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("CA file should exist: %v", err)
	}
}

func TestLoadOrCreateCA_Existing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	ca1, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("first LoadOrCreateCA: %v", err)
	}
	ca2, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("second LoadOrCreateCA: %v", err)
	}
	if !ca1.Certificate.Equal(ca2.Certificate) {
		t.Errorf("second call should load existing cert, not regenerate")
	}
	// 回归测试：重启后（第二次加载）私钥必须可用，否则无法签发 server 证书
	if ca2.PrivateKey == nil {
		t.Fatal("second call should load CA private key, got nil")
	}
}

// TestLoadOrCreateCA_CanSignAfterReload 验证 CA 重新加载后仍能签发证书
// （修复前 parseCAPEM 返回 PrivateKey=nil，导致 server 证书签发失败）。
func TestLoadOrCreateCA_CanSignAfterReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	if _, err := LoadOrCreateCA(path); err != nil {
		t.Fatalf("first LoadOrCreateCA: %v", err)
	}
	ca, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("second LoadOrCreateCA: %v", err)
	}
	if ca.PrivateKey == nil {
		t.Fatal("CA private key should be loaded after restart")
	}
	// 用重新加载的 CA 签发 server 证书，应成功
	if err := LoadOrCreateServerCert(ca, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"), "10.0.0.1", "box.lan"); err != nil {
		t.Fatalf("LoadOrCreateServerCert after reload: %v", err)
	}
}

// TestLoadOrCreateCA_RegeneratesWhenKeyMissing 验证旧版残留
// （有 ca.crt 无 ca.key）时重新生成 CA，而非返回无私钥的 CA。
func TestLoadOrCreateCA_RegeneratesWhenKeyMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	ca1, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("first LoadOrCreateCA: %v", err)
	}
	// 模拟旧版残留：删除 ca.key 但保留 ca.crt
	keyPath := caKeyPath(path)
	if err := os.Remove(keyPath); err != nil {
		t.Fatalf("remove ca.key: %v", err)
	}
	ca2, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("LoadOrCreateCA with missing key: %v", err)
	}
	if ca2.PrivateKey == nil {
		t.Fatal("should regenerate CA with private key when ca.key missing")
	}
	// 重新生成后证书应不同
	if ca1.Certificate.Equal(ca2.Certificate) {
		t.Error("should regenerate a new CA cert, not reuse the old one")
	}
}

func TestCAFileFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.crt")
	_, err := LoadOrCreateCA(path)
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ca: %v", err)
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		t.Fatalf("CA file should be PEM-encoded")
	}
	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Errorf("PEM block should be valid x509 cert: %v", err)
	}
}
