package auth

import (
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestIssueDeviceCert(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(dir + "/ca.crt")
	cert, key, err := IssueDeviceCert(ca, "test-device")
	if err != nil {
		t.Fatalf("IssueDeviceCert: %v", err)
	}
	if len(cert) == 0 {
		t.Error("cert should not be empty")
	}
	if len(key) == 0 {
		t.Error("key should not be empty")
	}
}

func TestIssueDeviceCert_Verify(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}
	certPEM, keyPEM, err := IssueDeviceCert(ca, "my-phone")
	if err != nil {
		t.Fatalf("IssueDeviceCert: %v", err)
	}

	// 解析证书
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		t.Fatal("invalid cert PEM")
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}

	// 验证证书由 CA 签发
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	if _, err := cert.Verify(opts); err != nil {
		t.Errorf("device cert should be valid: %v", err)
	}

	// 验证 Subject
	if cert.Subject.CommonName != "my-phone" {
		t.Errorf("CommonName = %q, want %q", cert.Subject.CommonName, "my-phone")
	}
	if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "mobilecoding" {
		t.Errorf("Organization should contain mobilecoding")
	}

	// 验证 KeyUsage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("cert should have KeyUsageDigitalSignature")
	}

	// 验证私钥可解析
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		t.Fatal("invalid key PEM")
	}
	priv, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	if priv.Curve != elliptic.P256() {
		t.Errorf("key curve should be P256")
	}
}

func TestIssueDeviceCert_NilCA(t *testing.T) {
	_, _, err := IssueDeviceCert(nil, "test")
	if err == nil {
		t.Error("expected error for nil CA")
	}
}

func TestIssueDeviceCert_NilCAKey(t *testing.T) {
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.crt")
	LoadOrCreateCA(caPath) // nolint:errcheck
	raw, _ := os.ReadFile(caPath)
	ca, _ := parseCAPEM(raw) // PrivateKey 为 nil
	_, _, err := IssueDeviceCert(ca, "test")
	if err == nil {
		t.Error("expected error for nil CA private key")
	}
}

func TestIssueDeviceCert_MultipleDevices(t *testing.T) {
	dir := t.TempDir()
	ca, _ := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))

	cert1, key1, err := IssueDeviceCert(ca, "device-a")
	if err != nil {
		t.Fatalf("IssueDeviceCert device-a: %v", err)
	}
	cert2, key2, err := IssueDeviceCert(ca, "device-b")
	if err != nil {
		t.Fatalf("IssueDeviceCert device-b: %v", err)
	}

	// 两个设备证书应不同
	if string(cert1) == string(cert2) {
		t.Error("different devices should produce different certs")
	}
	if string(key1) == string(key2) {
		t.Error("different devices should produce different keys")
	}

	// 验证两个证书都有效
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	for i, certPEM := range [][]byte{cert1, cert2} {
		block, _ := pem.Decode(certPEM)
		if block == nil {
			t.Fatalf("cert %d: invalid PEM", i)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatalf("cert %d: parse: %v", i, err)
		}
		opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
		if _, err := cert.Verify(opts); err != nil {
			t.Errorf("cert %d: should be valid: %v", i, err)
		}
	}
}

func TestIssueDeviceCert_TLSEndToEnd(t *testing.T) {
	dir := t.TempDir()
	ca, err := LoadOrCreateCA(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatalf("LoadOrCreateCA: %v", err)
	}

	// 生成设备证书
	certPEM, keyPEM, err := IssueDeviceCert(ca, "tls-device")
	if err != nil {
		t.Fatalf("IssueDeviceCert: %v", err)
	}

	// 加载设备 TLS 证书
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}

	// 验证证书链完整（设备证书可被 CA 验证）
	pool := x509.NewCertPool()
	pool.AddCert(ca.Certificate)
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("invalid cert PEM")
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}
	if _, err := parsedCert.Verify(opts); err != nil {
		t.Errorf("cert should verify: %v", err)
	}

	// 验证 tlsCert 可用
	if tlsCert.Certificate == nil {
		t.Error("tlsCert.Certificate should not be nil")
	}
}
