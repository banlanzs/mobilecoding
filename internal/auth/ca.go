package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CA 是内存中的 CA 证书与私钥。
type CA struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

// LoadOrCreateCA 加载 path 处的 CA 证书；若不存在则新建 RSA-2048 CA（10 年）。
// 父目录权限 0o700，文件权限 0o600。
func LoadOrCreateCA(path string) (*CA, error) {
	if raw, err := os.ReadFile(path); err == nil {
		return parseCAPEM(raw)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	return generateAndSaveCA(path)
}

func generateAndSaveCA(path string) (*CA, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate ca key: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:         "mytool",
			OrganizationalUnit: []string{"local"},
			Organization:       []string{"mytool dev CA"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create ca cert: %w", err)
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("mkdir ca dir: %w", err)
		}
		_ = os.Chmod(dir, 0o700)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write ca: %w", err)
	}
	_ = os.Chmod(path, 0o600)

	return &CA{Certificate: cert, PrivateKey: key}, nil
}

func parseCAPEM(raw []byte) (*CA, error) {
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("ca: invalid PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ca cert: %w", err)
	}
	if !cert.IsCA {
		return nil, errors.New("ca: cert is not a CA")
	}
	return &CA{Certificate: cert, PrivateKey: nil}, nil
}

// SignServerCSR 用 CA 私钥签发 server 证书（含 SAN: ip, dns, 127.0.0.1, localhost）。
func (c *CA) SignServerCSR(csr *x509.CertificateRequest, ip, dns string) ([]byte, error) {
	if c.PrivateKey == nil {
		return nil, errors.New("ca: private key not loaded")
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("invalid csr signature: %w", err)
	}
	serial, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip), net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{dns, "localhost"},
	}
	return x509.CreateCertificate(rand.Reader, tmpl, c.Certificate, csr.PublicKey, c.PrivateKey)
}
