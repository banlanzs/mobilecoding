// Package gateway 提供 mobilecoding HTTP 入口：healthz/version/SPA + REST + WS 升级。
package gateway

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/banlanzs/mobilecoding/internal/auth"
	"github.com/banlanzs/mobilecoding/internal/relay"
	"github.com/banlanzs/mobilecoding/internal/session"
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
	Relay       *relay.Server // Relay 中继服务器
}

func NewRouter(deps Dependencies, authToken string) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": deps.Version,
			"runtime": map[string]any{
				"defaultCommand": deps.DefaultCmd,
				"defaultArgs":    deps.DefaultArgs,
			},
		})
	})

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
	})

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

// claudeSettingsHandler 扫描 ~/.claude/settings.*.json 并返回配置列表。
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
