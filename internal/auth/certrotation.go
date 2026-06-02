package auth

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"time"
)

// CheckCertExpiry 检查证书是否在 30 天内过期。
// 返回 true 表示需要重新签发。
func CheckCertExpiry(certPath string) bool {
	raw, err := os.ReadFile(certPath)
	if err != nil {
		return true // 文件不存在，需要签发
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return true // 无效 PEM
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true // 无效证书
	}
	// 30 天内过期
	return time.Until(cert.NotAfter) < 30*24*time.Hour
}

// RotateServerCert 如果 server 证书过期则重新签发。
func RotateServerCert(ca *CA, certPath, keyPath, ip, dns string) (bool, error) {
	if !CheckCertExpiry(certPath) {
		return false, nil // 证书有效，不需要轮换
	}
	// 删除旧证书
	os.Remove(certPath)
	os.Remove(keyPath)
	// 重新签发
	if err := LoadOrCreateServerCert(ca, certPath, keyPath, ip, dns); err != nil {
		return false, err
	}
	return true, nil
}