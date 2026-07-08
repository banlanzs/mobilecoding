// mobilecoding 后端入口：装配 config + engine + session + gateway + ws，启动 HTTP 服务。
package main

import (
	"context"
	"crypto/tls"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/banlanzs/mobilecoding/internal/auth"
	"github.com/banlanzs/mobilecoding/internal/config"
	"github.com/banlanzs/mobilecoding/internal/engine"
	"github.com/banlanzs/mobilecoding/internal/gateway"
	"github.com/banlanzs/mobilecoding/internal/hook"
	"github.com/banlanzs/mobilecoding/internal/logx"
	"github.com/banlanzs/mobilecoding/internal/projection"
	"github.com/banlanzs/mobilecoding/internal/relay"
	"github.com/banlanzs/mobilecoding/internal/session"
	"github.com/banlanzs/mobilecoding/internal/store"
	"github.com/banlanzs/mobilecoding/internal/ws"
)

const version = "0.1.0"

//go:embed web/*
var webAssets embed.FS

func main() {
	// 子命令路由
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "server":
			// 显式 server 子命令：移除 "server" 参数，继续执行原有 server 逻辑
			os.Args = append(os.Args[:1], os.Args[2:]...)
		case "relay":
			runRelay(os.Args[2:])
			return
		case "claude":
			runClaude(os.Args[2:])
			return
		case "codex":
			runCodex(os.Args[2:])
			return
		case "version", "-version", "--version":
			fmt.Println(version)
			return
		case "help", "-help", "--help":
			printGlobalUsage()
			return
		}
	}

	// 无子命令或未识别参数 = 默认 server 行为（向后兼容）
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
		fmt.Fprintln(os.Stderr, "Usage: mobilecoding [flags]\n  -port               listen port (default 8443)\n  -ip                 local IP for cert & QR code (auto-detect if omitted)\n  -default-command    default AI command (claude|codex|opencode|aichat)\n  -default-args       default args for AI command (space-separated, quoted)\n  -launch-mode        managed|remote-control")
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

	// IP 选择：手动指定 > 自动检测
	localIP := cfg.IP
	if localIP == "" {
		localIP = auth.GetLocalIP()
	}
	// 显示所有可用 IP
	allIPs := auth.GetAllLocalIPs()
	if len(allIPs) > 1 {
		logger.Info("startup", "detected local IPs: %v (using %s)", allIPs, localIP)
	}
	if cfg.IP != "" {
		logger.Info("startup", "using manually specified IP: %s", cfg.IP)
	}

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
		IP:          firstNonEmpty(f.ip, env.IP),
		AuthToken:   firstNonEmpty(f.authToken, env.AuthToken),
		Workspace:   firstNonEmpty(f.workspace, env.Workspace),
		MTLS:        firstNonEmpty(f.mtls, env.MTLS),
		LogLevel:    firstNonEmpty(f.logLevel, env.LogLevel),
		DefaultCmd:  firstNonEmpty(f.defaultCmd, env.DefaultCmd),
		DefaultArgs: config.SplitArgs(os.ExpandEnv(firstNonEmpty(f.defaultArgs, env.DefaultArgs))),
		LaunchMode:  firstNonEmpty(f.launchMode, env.LaunchMode),
	}.WithDefaults()

	if c.AuthToken == "" {
		tok, err := config.NewToken()
		if err != nil {
			return c, fmt.Errorf("generate token: %w", err)
		}
		c.AuthToken = tok
		fmt.Fprintf(os.Stderr, "==> Generated new auth token (MVP 1: in-memory only): %s\n", tok)
	}
	if err := c.Validate(); err != nil {
		return c, err
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

	// 消息持久化存储
	msgStore, err := store.Open("")
	if err != nil {
		logger.Warn("startup", "open message store: %v (continuing without persistence)", err)
	} else {
		defer msgStore.Close()
		logger.Info("startup", "message store ready")
		// 启动时清理超过 7 天的旧消息
		if deleted, err := msgStore.CleanupOldSessions(7); err == nil && deleted > 0 {
			logger.Info("startup", "cleaned up %d old messages", deleted)
		}
	}

	// 创建 relay 服务器
	relayServer := relay.NewServer(relay.DefaultSessionConfig())

	// 会话元数据存储
	sessionStore, err := session.NewStore("")
	if err != nil {
		logger.Warn("startup", "open session store: %v (continuing without persistence)", err)
	} else {
		logger.Info("startup", "session store ready")
	}

	hub := ws.NewHub()
	mgr := session.NewManager()
	mgr.SetLogger(logger.Info)
	if sessionStore != nil {
		mgr.SetStore(sessionStore)
	}
	wsHandler := ws.NewHandler(hub, mgr, logger)

	// 权限 hook：Claude Code 的 PermissionRequest HTTP hook 端点
	serverCWD, _ := os.Getwd()
	hookRegistry := hook.NewRegistry()
	hookHandler := hook.NewHandler(hookRegistry, func(ev hook.Event) {
		// 把权限请求包装成 projection.Event 通过 hub 广播给 WS 客户端
		pe := projection.PermissionAskEventWithID(ev.SessionID, ev.RequestID, ev.ToolName, ev.ToolInputPrompt, ev.ToolInput)
		env, err := ws.ProjectionToEnvelope(pe)
		if err != nil {
			logger.Error("hook", "wrap event failed: %v", err)
			return
		}
		hub.Broadcast(env)
	})
	if cfg.LaunchMode == "remote-control" {
		hookHandler.AllowedCWD = serverCWD
	}
	hookHandler.Log = func(format string, args ...any) { logger.Info("hook-http", format, args...) }
	wsHandler.SetHookRegistry(hookRegistry)

	// 独立 HTTP 监听器（仅 127.0.0.1）服务 hook 端点，避开主端口的 HTTPS。
	// Claude CLI 的 HTTP POST 在 plain HTTP 上才能工作（无 cert 信任问题），且仅本地可达足够安全。
	hookListener, hookURL, err := startHookListener(cfg, hookHandler, logger)
	if err != nil {
		logger.Warn("startup", "start hook listener: %v (continue without)", err)
	} else {
		defer hookListener.Close()
		// 自动注入 hook 到当前项目 .claude/settings.local.json
		if err := installClaudeHook(cfg, hookURL, serverCWD, logger); err != nil {
			logger.Warn("startup", "install Claude hook: %v (continue without)", err)
		}
	}

	// 启动全局事件转发器：从 session.Manager 读取事件并广播到所有连接
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var saveCh chan<- saveRequest
	if msgStore != nil {
		saveCh = startAsyncWriter(ctx, msgStore, logger)
	}
	go forwardSessionEvents(ctx, mgr, hub, logger, saveCh)

	if cfg.LaunchMode == "remote-control" {
		if err := startNativeControlSession(ctx, cfg, mgr, logger); err != nil {
			return err
		}
	}

	r := gateway.NewRouter(gateway.Dependencies{
		FS:          staticFS,
		Version:     version,
		WS:          wsHandler,
		Session:     mgr,
		CA:          ca,
		DefaultCmd:  cfg.DefaultCmd,
		DefaultArgs: cfg.DefaultArgs,
		LaunchMode:  cfg.LaunchMode,
		Models:      cfg.Models,
		Relay:       relayServer,
		MsgStore:    msgStore,
	}, cfg.AuthToken)

	addr := ":" + cfg.Port
	logger.Info("startup", "listening on %s (mtls=%s), workspace=%s", addr, cfg.MTLS, cfg.Workspace)
	srv := &http.Server{Addr: addr, Handler: r, TLSConfig: tlsCfg}

	// 开发模式：同时启动一个纯 WS 监听器（绑定 0.0.0.0），供 Android 模拟器/真机调试用
	devPort := pickDevWSPort(cfg.Port)
	devAddr := "0.0.0.0:" + devPort
	devListener, err := net.Listen("tcp", devAddr)
	if err == nil {
		devSrv := &http.Server{Handler: r}
		go func() {
			logger.Info("startup", "dev ws listener (no TLS): %s", devListener.Addr().String())
			if err := devSrv.Serve(devListener); err != nil && err != http.ErrServerClosed {
				logger.Error("dev-ws", "serve: %v", err)
			}
		}()
		defer devListener.Close()
	} else {
		logger.Warn("startup", "dev ws listener skipped: %v", err)
	}

	return srv.ListenAndServeTLS("", "")
}

func startNativeControlSession(ctx context.Context, cfg config.Config, mgr *session.Manager, logger *logx.Logger) error {
	cwd, _ := os.Getwd()
	req := engine.ExecRequest{
		Command: cfg.DefaultCmd,
		Args:    cfg.DefaultArgs,
		CWD:     cwd,
		Cols:    120,
		Rows:    32,
	}
	run := engine.NewNativeRunner(req.Command)
	sid, err := mgr.Start(ctx, req, run)
	if err != nil {
		return fmt.Errorf("start native control session: %w", err)
	}
	logger.Info("startup", "native control session started: command=%s args=%v sessionId=%s cwd=%s", req.Command, req.Args, sid, req.CWD)
	return nil
}

// installClaudeHook 把 mobilecoding 的权限 hook 注入到当前项目本地 settings.local.json。
// 同时移除历史版本写入用户级 settings.json 的 mobilecoding hook，避免跨项目权限串线。
func installClaudeHook(cfg config.Config, hookURL, cwd string, logger *logx.Logger) error {
	if globalPath, err := hook.DefaultSettingsPath(); err == nil {
		if err := hook.NewSettingsInjector(globalPath).RemoveInstalledHook(); err != nil {
			logger.Warn("startup", "remove global Claude hook skipped: %v", err)
		}
	}

	path := hook.ProjectSettingsPath(cwd)
	inj := hook.NewSettingsInjector(path)
	if err := inj.Install(hook.HookConfig{
		URL:     hookURL,
		Token:   cfg.AuthToken,
		Timeout: 300,
	}); err != nil {
		logger.Warn("startup", "hook install skipped: %v", err)
		return nil
	}
	logger.Info("startup", "Claude hook installed: path=%s url=%s", path, hookURL)
	return nil
}

// startHookListener 启动独立 HTTP 监听器（仅绑定 127.0.0.1）服务 hook 端点。
// 端口优先级：MOBILECODING_HOOK_PORT 环境变量 > 主端口+1。
// 返回实际 listener 和对外 URL（含端口）。Bearer 鉴权与主服务器共用 cfg.AuthToken。
func startHookListener(cfg config.Config, h *hook.Handler, logger *logx.Logger) (net.Listener, string, error) {
	hookPort := pickHookPort(cfg.Port)
	addr := "127.0.0.1:" + hookPort
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", fmt.Errorf("listen %s: %w", addr, err)
	}
	// 构造带 Bearer 鉴权的子路由
	mux := http.NewServeMux()
	mux.Handle("/v1/hooks/permission-request", auth.BearerMiddleware(cfg.AuthToken, h))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 6 * time.Minute, // 比 hook 默认超时略长，确保响应能写回
	}
	go func() {
		logger.Info("startup", "hook http listener: %s", ln.Addr().String())
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			logger.Error("hook", "http serve: %v", err)
		}
	}()
	url := fmt.Sprintf("http://%s/v1/hooks/permission-request", ln.Addr().String())
	return ln, url, nil
}

// pickHookPort 决定 hook 监听端口：MOBILECODING_HOOK_PORT > 主端口+1。
func pickHookPort(mainPort string) string {
	if v := os.Getenv("MOBILECODING_HOOK_PORT"); v != "" {
		return v
	}
	n, err := strconv.Atoi(mainPort)
	if err != nil {
		return "8444"
	}
	return strconv.Itoa(n + 1)
}

// pickDevWSPort 决定开发 WS 监听端口：MOBILECODING_DEV_WS_PORT > 主端口+2。
func pickDevWSPort(mainPort string) string {
	if v := os.Getenv("MOBILECODING_DEV_WS_PORT"); v != "" {
		return v
	}
	n, err := strconv.Atoi(mainPort)
	if err != nil {
		return "8445"
	}
	return strconv.Itoa(n + 2)
}

// saveRequest 异步写入请求。
type saveRequest struct {
	sessionID string
	event     *projection.Event
}

// startAsyncWriter 启动异步消息写入 goroutine。
// 从 saveCh 读取请求，写入数据库后将 seq 回填到 event。
func startAsyncWriter(ctx context.Context, msgStore *store.MessageStore, logger *logx.Logger) chan<- saveRequest {
	ch := make(chan saveRequest, 512)
	go func() {
		for {
			select {
			case req := <-ch:
				seq, err := msgStore.SaveMessage(req.sessionID, *req.event)
				if err != nil {
					logger.Error("store", "async save failed: %v", err)
				} else {
					req.event.Seq = seq
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// forwardSessionEvents 从 session.Manager 读取事件并广播到所有 WebSocket 连接
func forwardSessionEvents(ctx context.Context, mgr *session.Manager, hub *ws.Hub, logger *logx.Logger, saveCh chan<- saveRequest) {
	input := mgr.Output()
	fwdCount := 0

	// phaseTracker 必须跨事件共享，否则 tool_start/tool_end/bash_start/bash_end/thinking_start/thinking_end
	// 等配对事件无法正确生成。session 切换时通过事件输入流自然重建（首个事件由新 phaseTracker 处理）。
	tracker := &projection.PhaseTracker{}

	for {
		select {
		case ev, ok := <-input:
			if !ok {
				logger.Debug("broadcast", "session output closed, forwarded %d events", fwdCount)
				return
			}
			// 每次都使用同一个 phaseTracker，跨事件保持状态
			sid := mgr.SessionID()
			projEvents := projection.Project([]engine.Event{ev}, sid, tracker)
			logger.Debug("broadcast", "event kind=%s projected=%d", ev.Kind, len(projEvents))

			for _, pe := range projEvents {
				// 异步持久化消息，seq 由 writer goroutine 分配后回填
				if saveCh != nil {
					saveCh <- saveRequest{sessionID: sid, event: &pe}
				}
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

// printGlobalUsage 打印全局帮助信息，显示所有可用子命令。
func printGlobalUsage() {
	fmt.Fprintf(os.Stderr, "Usage: mobilecoding [command] [flags]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  (default)       Start mobilecoding server (same as 'server' subcommand)\n")
	fmt.Fprintf(os.Stderr, "  server          Start mobilecoding server with HTTPS and WebSocket\n")
	fmt.Fprintf(os.Stderr, "  relay           Connect to relay server as agent\n")
	fmt.Fprintf(os.Stderr, "  claude          Start Claude Code + server (remote control mode)\n")
	fmt.Fprintf(os.Stderr, "  codex           Start Codex + server (remote control mode)\n")
	fmt.Fprintf(os.Stderr, "  version         Print version and exit\n")
	fmt.Fprintf(os.Stderr, "  help            Print this help message\n")
	fmt.Fprintf(os.Stderr, "\nServer Flags:\n")
	fmt.Fprintf(os.Stderr, "  -port <port>               Listen port (default: 8443)\n")
	fmt.Fprintf(os.Stderr, "  -ip <ip>                   Local IP for cert & QR code (auto-detect if omitted)\n")
	fmt.Fprintf(os.Stderr, "  -workspace <path>          Workspace root directory\n")
	fmt.Fprintf(os.Stderr, "  -auth-token <token>        Auth token for API access\n")
	fmt.Fprintf(os.Stderr, "  -mtls <mode>               mTLS mode: none|optional|required\n")
	fmt.Fprintf(os.Stderr, "  -log-level <level>         Log level: debug|info|warn|error\n")
	fmt.Fprintf(os.Stderr, "  -default-command <cmd>     Default AI command (claude|codex|opencode|aichat)\n")
	fmt.Fprintf(os.Stderr, "  -default-args <args>       Default args for AI command\n")
	fmt.Fprintf(os.Stderr, "  -launch-mode <mode>        Launch mode: managed|remote-control\n")
	fmt.Fprintf(os.Stderr, "\nRelay Flags:\n")
	fmt.Fprintf(os.Stderr, "  -relay <url>               Relay server URL (wss://...)\n")
	fmt.Fprintf(os.Stderr, "  -session <id>              Session ID (optional)\n")
	fmt.Fprintf(os.Stderr, "  -insecure                  Skip TLS verification\n")
	fmt.Fprintf(os.Stderr, "\nClaude/Codex Flags:\n")
	fmt.Fprintf(os.Stderr, "  -settings <path>           Claude settings file (claude only)\n")
	fmt.Fprintf(os.Stderr, "  -model <model>             Model override (claude only)\n")
	fmt.Fprintf(os.Stderr, "  -port <port>               Server port (default: 8443)\n")
	fmt.Fprintf(os.Stderr, "  -resume <id>               Resume session ID (claude only)\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  mobilecoding                              # Start server (default)\n")
	fmt.Fprintf(os.Stderr, "  mobilecoding server -port 9000            # Start server on custom port\n")
	fmt.Fprintf(os.Stderr, "  mobilecoding claude --model opus          # Start Claude with Opus model\n")
	fmt.Fprintf(os.Stderr, "  mobilecoding relay -relay wss://...       # Connect to relay server\n")
}
