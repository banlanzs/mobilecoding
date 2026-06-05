// Package gateway 提供 mobilecoding HTTP 入口：healthz/version/SPA + REST + WS 升级。
package gateway

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/banlanzs/mobilecoding/internal/auth"
	"github.com/banlanzs/mobilecoding/internal/relay"
	"github.com/banlanzs/mobilecoding/internal/session"
	"github.com/banlanzs/mobilecoding/internal/store"
	"github.com/banlanzs/mobilecoding/internal/ws"
)

type Dependencies struct {
	FS          fs.FS
	Version     string
	WS          *ws.Handler
	Session     *session.Manager
	Workspace   string   // 用于 skill 列表
	StoreDir    string   // 用于 memory 读写
	CA          *auth.CA // 用于设备证书签发
	DefaultCmd  string
	DefaultArgs []string
	Models      string           // 逗号分隔: label:value,...
	Relay       *relay.Server    // Relay 中继服务器
	MsgStore    *store.MessageStore // 消息持久化存储（可选）
	// HookHandler 已移至独立 HTTP 监听器（startHookListener），不通过主 router 提供。
}

func NewRouter(deps Dependencies, authToken string) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/api/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		raw := deps.Models
		// 如果传了 ?settings= 参数，从对应 settings 文件中读取模型配置
		if settingsPath := r.URL.Query().Get("settings"); settingsPath != "" {
			if custom := readSettingsModels(settingsPath); custom != "" {
				raw = custom
			}
		}
		if raw == "" {
			raw = "默认模型:,Sonnet 4.6:claude-sonnet-4-6,Opus 4.8:claude-opus-4-8,Haiku 4.5:claude-haiku-4-5"
		}
		models := parseModels(raw)
		_ = json.NewEncoder(w).Encode(models)
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		cwd, _ := os.Getwd()
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": deps.Version,
			"runtime": map[string]any{
				"defaultCommand": deps.DefaultCmd,
				"defaultArgs":    deps.DefaultArgs,
				"cwd":            cwd,
			},
		})
	})

	// 客户端状态端点（不需要认证，供 mc CLI 轮询检测手机连接）
	r.Get("/api/v1/clients", clientsHandler(deps.WS))
	r.Get("/api/v1/session-id", sessionIDHandler(deps.Session))

	r.With(func(next http.Handler) http.Handler {
		return auth.BearerMiddleware(authToken, next)
	}).Handle("/api/v1/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if deps.WS == nil {
			http.Error(w, "ws handler not configured", http.StatusServiceUnavailable)
			return
		}
		c, err := ws.NewConn(w, r)
		if err != nil {
			http.Error(w, "ws upgrade failed", http.StatusBadRequest)
			return
		}
		_ = deps.WS.ServeConn(r.Context(), c)
	}))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return auth.BearerMiddleware(authToken, next)
		})
		r.Get("/skills", skillsHandler(deps.Workspace))
		r.Get("/memory", memoryListHandler(deps.StoreDir))
		r.Put("/memory/{name}", memoryUpdateHandler(deps.StoreDir))
		r.Post("/device-cert", deviceCertHandler(deps.CA))
		r.Get("/claude-settings", claudeSettingsHandler())
		r.Get("/hook-status", hookStatusHandler())
		r.Get("/messages", messagesHandler(deps.MsgStore))
		r.Post("/resume", resumeHandler(deps.Session, deps.WS))
	})

	// Claude Code HTTP hook 端点已移至独立 HTTP 监听器（startHookListener），
	// 不再挂在主 router 上 —— 主端口是 HTTPS + 客户端证书/Token 鉴权，Claude CLI 无 cert。

	// Relay 中继端点（不需要认证，使用配对码认证）
	if deps.Relay != nil {
		relayHandler := deps.Relay.Handler()
		r.Handle("/relay/*", http.StripPrefix("/relay", relayHandler))
	}

	if deps.FS != nil {
		r.Handle("/*", spaHandler(deps.FS))
	}

	return r
}

// hookStatusHandler 返回 hook 注入状态（settings.json 是否包含 mobilecoding 标记的 PermissionRequest hook）。
// 便于手机端排查权限弹窗问题：返回 { installed, settingsPath, hookURL }。
func hookStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		home, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, "cannot determine home dir", http.StatusInternalServerError)
			return
		}
		settingsPath := filepath.Join(home, ".claude", "settings.json")
		type hookInfo struct {
			EventName   string `json:"eventName"`
			URL         string `json:"url"`
			HasMobilec  bool   `json:"hasMobilecodingMarker"`
			TokenPrefix string `json:"tokenPrefix"`
		}
		type status struct {
			Installed    bool       `json:"installed"`
			SettingsPath string     `json:"settingsPath"`
			SettingsErr  string     `json:"settingsError,omitempty"`
			HookURL      string     `json:"hookUrl,omitempty"`
			Hooks        []hookInfo `json:"hooks"`
		}
		out := status{SettingsPath: settingsPath}
		data, err := os.ReadFile(settingsPath)
		if err != nil {
			if os.IsNotExist(err) {
				out.SettingsErr = "settings.json not found"
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(out)
				return
			}
			out.SettingsErr = err.Error()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
			return
		}
		var settings map[string]any
		if err := json.Unmarshal(data, &settings); err != nil {
			out.SettingsErr = "parse error: " + err.Error()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
			return
		}
		hooks, _ := settings["hooks"].(map[string]any)
		if hooks == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(out)
			return
		}
		for eventName, raw := range hooks {
			list, _ := raw.([]any)
			for _, item := range list {
				m, _ := item.(map[string]any)
				if m == nil {
					continue
				}
				hs, _ := m["hooks"].([]any)
				for _, h := range hs {
					hm, _ := h.(map[string]any)
					if hm == nil {
						continue
					}
					if hm["_mobilecoding"] != "mobilecoding-hook" {
						continue
					}
					out.Installed = true
					url, _ := hm["url"].(string)
					out.HookURL = url
					headers, _ := hm["headers"].(map[string]any)
					token, _ := headers["Authorization"].(string)
					tokenPrefix := ""
					if strings.HasPrefix(token, "Bearer ") {
						tokenPrefix = "Bearer " + token[len("Bearer "):min(8, len(token)-7)]
					}
					out.Hooks = append(out.Hooks, hookInfo{
						EventName:   eventName,
						URL:         url,
						HasMobilec:  true,
						TokenPrefix: tokenPrefix,
					})
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
// 返回格式：[{ name: "axonhub", path: "C:/Users/xxx/.claude/settings.axonhub.json" }, ...]
func claudeSettingsHandler() http.HandlerFunc {
	type settingEntry struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		home, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, "cannot determine home dir", http.StatusInternalServerError)
			return
		}
		claudeDir := filepath.Join(home, ".claude")
		entries, err := os.ReadDir(claudeDir)
		if err != nil {
			// .claude 目录不存在，返回空列表
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]settingEntry{})
			return
		}

		var settings []settingEntry
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			// 匹配 settings.*.json（排除 settings.json）
			if strings.HasPrefix(name, "settings.") && strings.HasSuffix(name, ".json") && name != "settings.json" {
				profileName := strings.TrimSuffix(strings.TrimPrefix(name, "settings."), ".json")
				settings = append(settings, settingEntry{
					Name: profileName,
					Path: filepath.Join(claudeDir, name),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(settings)
	}
}

// readSettingsModels 从 settings JSON 文件中提取模型配置。
// 支持三种格式：
//   - "models" 字段：逗号分隔的 label:value 列表
//   - env 中的 ANTHROPIC_*_MODEL 变量（Haiku/Sonnet/Opus/默认）
func readSettingsModels(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return ""
	}
	// 优先：显式 "models" 字段
	if modelsRaw, ok := m["models"].(string); ok && modelsRaw != "" {
		return modelsRaw
	}
	// 从 env 中提取所有 ANTHROPIC_*_MODEL
	if env, ok := m["env"].(map[string]any); ok {
		modelKeys := []struct{ key, label string }{
			{"ANTHROPIC_DEFAULT_HAIKU_MODEL", "Haiku"},
			{"ANTHROPIC_DEFAULT_SONNET_MODEL", "Sonnet"},
			{"ANTHROPIC_DEFAULT_OPUS_MODEL", "Opus"},
			{"ANTHROPIC_MODEL", "默认"},
		}
		var parts []string
		seen := map[string]bool{}
		for _, mk := range modelKeys {
			if v, ok := env[mk.key].(string); ok && v != "" && !seen[v] {
				seen[v] = true
				parts = append(parts, mk.label+":"+v)
			}
		}
		if len(parts) > 0 {
			// 确保 "默认模型" (空值) 在最前面
			return "默认模型:," + strings.Join(parts, ",")
		}
	}
	return ""
}

// parseModels 解析 "label:value,label:value" 格式的模型列表。
func parseModels(raw string) []map[string]string {
	var models []map[string]string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// 支持 "label:value" 和 "label:" (空值) 两种格式
		idx := strings.Index(part, ":")
		if idx < 0 {
			models = append(models, map[string]string{"label": part, "value": part})
		} else {
			models = append(models, map[string]string{
				"label": part[:idx],
				"value": part[idx+1:],
			})
		}
	}
	if len(models) == 0 {
		models = append(models, map[string]string{"label": "默认模型", "value": ""})
	}
	return models
}

// messagesHandler 返回历史消息查询 API。
// GET /api/v1/messages?session_id=xxx&after_seq=0&limit=100
// GET /api/v1/messages?session_id=xxx&before_seq=999&limit=100
func messagesHandler(msgStore *store.MessageStore) http.HandlerFunc {
	type response struct {
		Messages []store.SequencedMessage `json:"messages"`
		HasMore  bool                     `json:"hasMore"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if msgStore == nil {
			http.Error(w, "message store not available", http.StatusServiceUnavailable)
			return
		}
		sessionID := r.URL.Query().Get("session_id")
		if sessionID == "" {
			http.Error(w, "session_id required", http.StatusBadRequest)
			return
		}
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		afterStr := r.URL.Query().Get("after_seq")
		beforeStr := r.URL.Query().Get("before_seq")

		var msgs []store.SequencedMessage
		var err error
		switch {
		case afterStr != "" && beforeStr != "":
			http.Error(w, "after_seq and before_seq are mutually exclusive", http.StatusBadRequest)
			return
		case afterStr != "":
			afterSeq, e := strconv.ParseInt(afterStr, 10, 64)
			if e != nil {
				http.Error(w, "invalid after_seq", http.StatusBadRequest)
				return
			}
			msgs, err = msgStore.GetMessagesAfter(sessionID, afterSeq, limit)
		case beforeStr != "":
			beforeSeq, e := strconv.ParseInt(beforeStr, 10, 64)
			if e != nil {
				http.Error(w, "invalid before_seq", http.StatusBadRequest)
				return
			}
			msgs, err = msgStore.GetMessagesBefore(sessionID, beforeSeq, limit)
		default:
			msgs, err = msgStore.GetMessagesAfter(sessionID, 0, limit)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hasMore := len(msgs) > limit
		if hasMore {
			msgs = msgs[:limit]
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response{Messages: msgs, HasMore: hasMore})
	}
}

// clientsHandler 返回当前 WebSocket 客户端连接数。
// 供 CLI 包装器轮询检测手机连接状态。
func clientsHandler(wsHandler *ws.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		count := 0
		if wsHandler != nil {
			count = wsHandler.SubscriberCount()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"subscribers": count})
	}
}

// sessionIDHandler 返回当前活跃会话的 Claude resume session ID。
// 供 mc CLI 在模式切换时获取 --resume 参数。
func sessionIDHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if mgr == nil {
			http.Error(w, "session manager not available", http.StatusServiceUnavailable)
			return
		}
		resumeID := mgr.ResumeSessionID()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"sessionId":        mgr.SessionID(),
			"resumeSessionId":  resumeID,
		})
	}
}

// resumeHandler 接收 mc CLI 发来的 resume session ID，停止当前会话并存储 resume ID。
// 手机端下次调用 session.start 时会自动使用此 resume ID 继续会话。
func resumeHandler(mgr *session.Manager, wsHandler *ws.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p struct {
			ResumeSessionID string `json:"resumeSessionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.ResumeSessionID == "" {
			http.Error(w, "resumeSessionId required", http.StatusBadRequest)
			return
		}
		// 停止当前会话（如果有）
		if mgr.SessionID() != "" {
			mgr.Stop()
		}
		// 存储 resume ID，供下次 session.start 使用
		wsHandler.SetPendingResumeID(p.ResumeSessionID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "resumeSessionId": p.ResumeSessionID})
	}
}

func deviceCertHandler(ca *auth.CA) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ca == nil || ca.PrivateKey == nil {
			http.Error(w, "CA private key not available", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			DeviceName string `json:"device_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if req.DeviceName == "" {
			http.Error(w, "device_name required", http.StatusBadRequest)
			return
		}
		certPEM, keyPEM, err := auth.IssueDeviceCert(ca, req.DeviceName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"cert": string(certPEM),
			"key":  string(keyPEM),
		})
	}
}
