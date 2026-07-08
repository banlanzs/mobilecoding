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

// LoadOrCreateCA 加载 path 处的 CA 证书和配套私钥；若任一不存在则新建 RSA-2048 CA（10 年）。
// 私钥保存在 path 同目录的 ca.key（去掉原扩展名加 .key）。
// 父目录权限 0o700，文件权限 0o600。
func LoadOrCreateCA(path string) (*CA, error) {
	keyPath := caKeyPath(path)
	certRaw, certErr := os.ReadFile(path)
	keyRaw, keyErr := os.ReadFile(keyPath)
	if certErr == nil && keyErr == nil {
		// 证书和私钥都存在，加载两者
		ca, err := parseCAPEM(certRaw)
		if err != nil {
			return nil, fmt.Errorf("parse ca cert: %w", err)
		}
		key, err := parseRSAPrivateKeyPEM(keyRaw)
		if err != nil {
			return nil, fmt.Errorf("parse ca key: %w", err)
		}
		ca.PrivateKey = key
		return ca, nil
	}
	// 任一文件缺失（含旧版残留：有 ca.crt 无 ca.key）→ 重新生成 CA
	// 用原始读错误报告非"不存在"的故障
	if certErr != nil && !os.IsNotExist(certErr) {
		return nil, fmt.Errorf("read ca: %w", certErr)
	}
	if keyErr != nil && !os.IsNotExist(keyErr) {
		return nil, fmt.Errorf("read ca key: %w", keyErr)
	}
	return generateAndSaveCA(path)
}

// caKeyPath 由 ca.crt 路径推导 ca.key 路径（替换扩展名）。
func caKeyPath(caPath string) string {
	ext := filepath.Ext(caPath)
	return caPath[:len(caPath)-len(ext)] + ".key"
}

func generateAndSaveCA(path string) (*CA, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate ca key: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName:         "mobilecoding",
			OrganizationalUnit: []string{"local"},
			Organization:       []string{"mobilecoding dev CA"},
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

	// 私钥落盘，供重启后加载（修复私钥只在内存、进程重启后无法签发证书的问题）
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal ca key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	keyPath := caKeyPath(path)
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("write ca key: %w", err)
	}
	_ = os.Chmod(keyPath, 0o600)

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

// parseRSAPrivateKeyPEM 解析 PEM 编码的 RSA 私钥（PKCS8 或 PKCS1）。
func parseRSAPrivateKeyPEM(raw []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("ca key: invalid PEM block")
	}
	var key any
	var err error
	switch block.Type {
	case "PRIVATE KEY":
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("ca key: unexpected PEM type %q", block.Type)
	}
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("ca key: not an RSA private key")
	}
	return rsaKey, nil
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
