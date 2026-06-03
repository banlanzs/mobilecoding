// mobilecoding 后端入口：装配 config + engine + session + gateway + ws，启动 HTTP 服务。
package main

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/banlanzs/mobilecoding/internal/auth"
	"github.com/banlanzs/mobilecoding/internal/config"
	"github.com/banlanzs/mobilecoding/internal/engine"
	"github.com/banlanzs/mobilecoding/internal/gateway"
	"github.com/banlanzs/mobilecoding/internal/logx"
	"github.com/banlanzs/mobilecoding/internal/projection"
	"github.com/banlanzs/mobilecoding/internal/session"
	"github.com/banlanzs/mobilecoding/internal/ws"
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
		fmt.Fprintln(os.Stderr, "Usage: mobilecoding [flags]\n  -port               listen port (default 8443)\n  -default-command    default AI command (claude|codex|opencode|aichat)\n  -default-args       default args for AI command (space-separated, quoted)")
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

	// 日志写入文件 + 清理旧日志
	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".mobilecoding", "logs")
	logFile, err := logx.OpenLogFile(logDir)
	if err == nil {
		logger = logx.NewWithMultiWriter(os.Stderr, logFile)
		logger.SetLevel(parseLevel(cfg.LogLevel))
		_ = logx.CleanOldLogs(logDir, 7)
	}

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
	localIP := auth.GetLocalIP()
	if err := auth.LoadOrCreateServerCert(ca, certPath, keyPath, localIP, "localhost"); err != nil {
		logger.Error("startup", "load server cert: %v", err)
		os.Exit(1)
	}
	// 证书过期检查 + 轮换
	rotated, err := auth.RotateServerCert(ca, certPath, keyPath, localIP, "localhost")
	if err != nil {
		logger.Error("startup", "rotate cert: %v", err)
		os.Exit(1)
	}
	if rotated {
		logger.Info("startup", "server cert rotated (expired or expiring soon)")
	}
	logger.Info("startup", "TLS ready: ca=%s cert=%s key=%s", caPath, certPath, keyPath)

	tlsCfg, err := buildTLSConfig(cfg.MTLS, ca, certPath, keyPath)
	if err != nil {
		logger.Error("startup", "build TLS config: %v", err)
		os.Exit(1)
	}

	// 打印二维码供客户端扫码配对
	if err := auth.PrintQRCode(fmt.Sprintf("https://%s:%s/?token=%s", localIP, cfg.Port, cfg.AuthToken)); err != nil {
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
		Port:        firstNonEmpty(f.port, env.Port, "8443"),
		AuthToken:   firstNonEmpty(f.authToken, env.AuthToken),
		Workspace:   firstNonEmpty(f.workspace, env.Workspace),
		MTLS:        firstNonEmpty(f.mtls, env.MTLS),
		LogLevel:    firstNonEmpty(f.logLevel, env.LogLevel),
		DefaultCmd:  firstNonEmpty(f.defaultCmd, env.DefaultCmd),
		DefaultArgs: config.SplitArgs(os.ExpandEnv(firstNonEmpty(f.defaultArgs, env.DefaultArgs))),
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

func handleConfigReload(cfg *config.Config, logger *logx.Logger) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for range sig {
			newCfg, err := config.Load()
			if err != nil {
				logger.Error("reload", "failed to reload config: %v", err)
				continue
			}
			// 更新可热重载的配置
			cfg.LogLevel = newCfg.LogLevel
			logger.SetLevel(parseLevel(cfg.LogLevel))
			logger.Info("reload", "config reloaded (log level: %s)", cfg.LogLevel)
		}
	}()
}

func run(cfg config.Config, logger *logx.Logger, tlsCfg *tls.Config, ca *auth.CA) error {
	// 启动配置热重载（SIGHUP）
	handleConfigReload(&cfg, logger)

	staticFS, err := fs.Sub(webAssets, "web")
	if err != nil {
		return fmt.Errorf("embed web: %w", err)
	}
	if _, err := fs.Stat(staticFS, "."); err != nil {
		logger.Warn("startup", "embedded web/ missing; using stub SPA")
	}

	hub := ws.NewHub()
	mgr := session.NewManager()
	mgr.SetLogger(logger.Info)
	wsHandler := ws.NewHandler(hub, mgr, logger)

	// 启动全局事件转发器：从 session.Manager 读取事件并广播到所有连接
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go forwardSessionEvents(ctx, mgr, hub, logger)

	r := gateway.NewRouter(gateway.Dependencies{
		FS:          staticFS,
		Version:     version,
		WS:          wsHandler,
		Session:     mgr,
		CA:          ca,
		DefaultCmd:  cfg.DefaultCmd,
		DefaultArgs: cfg.DefaultArgs,
	}, cfg.AuthToken)

	addr := ":" + cfg.Port
	logger.Info("startup", "listening on %s (mtls=%s), workspace=%s", addr, cfg.MTLS, cfg.Workspace)
	srv := &http.Server{Addr: addr, Handler: r, TLSConfig: tlsCfg}
	return srv.ListenAndServeTLS("", "")
}

// forwardSessionEvents 从 session.Manager 读取事件并广播到所有 WebSocket 连接
func forwardSessionEvents(ctx context.Context, mgr *session.Manager, hub *ws.Hub, logger *logx.Logger) {
	input := mgr.Output()
	fwdCount := 0

	for {
		select {
		case ev, ok := <-input:
			if !ok {
				logger.Debug("broadcast", "session output closed, forwarded %d events", fwdCount)
				return
			}
			// 注意：这里不做 projection，因为每个 handler 会自己做 projection
			// 直接将 engine.Event 包装成某种可序列化的格式
			// 但是为了保持一致性，我们在这里做 projection
			sid := mgr.SessionID()
			projEvents := projection.Project([]engine.Event{ev}, sid)
			logger.Debug("broadcast", "event kind=%s projected=%d", ev.Kind, len(projEvents))

			for _, pe := range projEvents {
				env, err := ws.ProjectionToEnvelope(pe)
				if err != nil {
					logger.Error("broadcast", "projectionToEnvelope failed: %v", err)
					continue
				}
				hub.Broadcast(env)
				fwdCount++
				if fwdCount <= 10 || fwdCount%50 == 0 {
					logger.Debug("broadcast", "broadcasted envelope #%d type=%s to %d subscribers", fwdCount, pe.Type, hub.SubscriberCount())
				}
			}
		case <-ctx.Done():
			logger.Debug("broadcast", "context cancelled, forwarded %d events", fwdCount)
			return
		}
	}
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
