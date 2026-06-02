package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/banlanzs/mobilecoding/internal/auth"
)

// buildTLSConfig 根据 mtls 模式构造 *tls.Config。
//   - optional: 强制 HTTPS，不要求客户端证书
//   - required: 强制 HTTPS + 客户端证书
//   - none: 返回错误（MVP 1.1 不支持明文）
func buildTLSConfig(mtlsMode string, ca *auth.CA, certPath, keyPath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("load server keypair: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	switch mtlsMode {
	case "optional", "":
		cfg.ClientAuth = tls.NoClientCert
	case "required":
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		if ca != nil {
			pool := x509.NewCertPool()
			pool.AddCert(ca.Certificate)
			cfg.ClientCAs = pool
		}
	case "none":
		return nil, errors.New("mtls=none is not supported in MVP 1.1")
	default:
		return nil, fmt.Errorf("invalid mtls mode: %q", mtlsMode)
	}
	return cfg, nil
}
