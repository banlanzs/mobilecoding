// mytool 后端入口：装配 config + engine + session + gateway + ws，启动 HTTP 服务。
package main

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jaycrl/mytool/internal/auth"
	"github.com/jaycrl/mytool/internal/config"
	"github.com/jaycrl/mytool/internal/gateway"
	"github.com/jaycrl/mytool/internal/logx"
	"github.com/jaycrl/mytool/internal/session"
	"github.com/jaycrl/mytool/internal/ws"
)

const version = "0.1.0"

//go:embed web/*
var webAssets embed.FS

func main() {
	flags, err := parseServerFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flag parse: %v\n", err)
		os.Exit(2)
	}
	if flags.showVersion {
		fmt.Println(version)
		return
	}
	if flags.showHelp {
		fmt.Fprintln(os.Stderr, "Usage: mytool [flags]\n  -port          listen port (default 8443)")
		return
	}

	// 1. Build config first (so env vars are merged)
	cfg, err := buildConfig(flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	// 2. Create logger with the merged log level
	logger := logx.New()
	logger.SetLevel(parseLevel(cfg.LogLevel))

	// 3. TLS 准备：生成/加载 CA + server 证书
	caDir := cfg.AuthDir
	caPath := filepath.Join(caDir, "ca.crt")
	ca, err := auth.LoadOrCreateCA(caPath)
	if err != nil {
		logger.Error("startup", "load CA: %v", err)
		os.Exit(1)
	}
	certPath := filepath.Join(caDir, "server.crt")
	keyPath := filepath.Join(caDir, "server.key")
	if err := auth.LoadOrCreateServerCert(ca, certPath, keyPath, "127.0.0.1", "localhost"); err != nil {
		logger.Error("startup", "load server cert: %v", err)
		os.Exit(1)
	}
	logger.Info("startup", "TLS ready: ca=%s cert=%s key=%s", caPath, certPath, keyPath)

	tlsCfg, err := buildTLSConfig(cfg.MTLS, ca, certPath, keyPath)
	if err != nil {
		logger.Error("startup", "build TLS config: %v", err)
		os.Exit(1)
	}

	// 打印二维码供客户端扫码配对
	if err := auth.PrintQRCode(fmt.Sprintf("https://%s:%s/?token=%s", auth.GetLocalIP(), cfg.Port, cfg.AuthToken)); err != nil {
		logger.Warn("startup", "print QR code failed: %v", err)
	}

	// 4. Run
	if err := run(cfg, logger, tlsCfg, ca); err != nil {
		logger.Error("startup", "run: %v", err)
		os.Exit(1)
	}
}

func buildConfig(f serverFlags) (config.Config, error) {
	env := config.FromEnv()
	c := config.Config{
		Port:       firstNonEmpty(f.port, env.Port, "8443"),
		AuthToken:  firstNonEmpty(f.authToken, env.AuthToken),
		Workspace:  firstNonEmpty(f.workspace, env.Workspace),
		MTLS:       firstNonEmpty(f.mtls, env.MTLS),
		LogLevel:   firstNonEmpty(f.logLevel, env.LogLevel),
		DefaultCmd: firstNonEmpty(f.defaultCmd, env.DefaultCmd),
	}.WithDefaults()

	if c.AuthToken == "" {
		tok, err := config.NewToken()
		if err != nil {
			return c, fmt.Errorf("generate token: %w", err)
		}
		c.AuthToken = tok
		fmt.Fprintf(os.Stderr, "==> Generated new auth token (MVP 1: in-memory only): %s\n", tok)
	}

	if err := os.MkdirAll(c.Workspace, 0o755); err != nil {
		return c, fmt.Errorf("create workspace: %w", err)
	}
	return c, nil
}

func run(cfg config.Config, logger *logx.Logger, tlsCfg *tls.Config, ca *auth.CA) error {
	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("embed web: %w", err)
	}
	if _, err := fs.Stat(staticFS, "."); err != nil {
		logger.Warn("startup", "embedded web/ missing; using stub SPA")
	}

	hub := ws.NewHub()
	mgr := session.NewManager()
	wsHandler := ws.NewHandler(hub, mgr)

	r := gateway.NewRouter(gateway.Dependencies{
		FS:      staticFS,
		Version: version,
		WS:      wsHandler,
		Session: mgr,
		CA:      ca,
	}, cfg.AuthToken)

	addr := ":" + cfg.Port
	logger.Info("startup", "listening on %s (mtls=%s), workspace=%s", addr, cfg.MTLS, cfg.Workspace)
	srv := &http.Server{Addr: addr, Handler: r, TLSConfig: tlsCfg}
	return srv.ListenAndServeTLS("", "")
}

func parseLevel(s string) logx.Level {
	switch s {
	case "debug":
		return logx.LevelDebug
	case "warn":
		return logx.LevelWarn
	case "error":
		return logx.LevelError
	}
	return logx.LevelInfo
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

var _ = filepath.Join
var _ = context.Background
