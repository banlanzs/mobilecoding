package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// LoadOrCreateServerCert 若 path 已存在且 SAN 包含 ip 则跳过；否则用 ca 签发新 server 证书。
// 证书含 SAN: <ip>, <dns>, 127.0.0.1, localhost（由 ca.SignServerCSR 注入）。
// Key 用 ECDSA P-256，文件 0o600。
func LoadOrCreateServerCert(ca *CA, certPath, keyPath, ip, dns string) error {
	if _, err := os.Stat(certPath); err == nil && certHasIPSAN(certPath, ip) {
		return nil
	}
	// 旧证书 SAN 不包含新 IP → 删除后重签
	if _, err := os.Stat(certPath); err == nil {
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}
	if ca == nil || ca.PrivateKey == nil {
		return errors.New("servercert: CA private key is required to sign new server cert")
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate server key: %w", err)
	}
	csrTmpl := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         "mytool",
			OrganizationalUnit: []string{"server"},
			Organization:       []string{dns},
		},
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, csrTmpl, priv)
	if err != nil {
		return fmt.Errorf("create csr: %w", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return fmt.Errorf("parse csr: %w", err)
	}
	der, err := ca.SignServerCSR(csr, ip, dns)
	if err != nil {
		return fmt.Errorf("sign server cert: %w", err)
	}
	if err := writePEMFile(certPath, "CERTIFICATE", der); err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return fmt.Errorf("marshal server key: %w", err)
	}
	return writePEMFile(keyPath, "EC PRIVATE KEY", keyDER)
}

// certHasIPSAN 检查 certPath 处的证书 SAN 是否包含 ip。
func certHasIPSAN(certPath, ip string) bool {
	raw, err := os.ReadFile(certPath)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	target := net.ParseIP(ip)
	if target == nil {
		return false
	}
	for _, sanIP := range cert.IPAddresses {
		if sanIP.Equal(target) {
			return true
		}
	}
	return false
}

// parseAndVerifyCert 解析 PEM 并用 pool 验证。测试 helper。
func parseAndVerifyCert(pemBytes []byte, pool *x509.CertPool) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("servercert: invalid PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	opts := x509.VerifyOptions{Roots: pool, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	if _, err := cert.Verify(opts); err != nil {
		return nil, err
	}
	return cert, nil
}

// writePEMFile 写 PEM-encoded 内容到 path，父目录 0o700，文件 0o600。
func writePEMFile(path, blockType string, der []byte) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return os.Chmod(path, 0o600)
}
