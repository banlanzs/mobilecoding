// Package gateway 提供 mytool HTTP 入口：healthz/version/SPA + REST 占位 + WS 升级。
package gateway

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jaycrl/mytool/internal/auth"
	"github.com/jaycrl/mytool/internal/session"
	"github.com/jaycrl/mytool/internal/ws"
)

type Dependencies struct {
	FS        fs.FS
	Version   string
	WS        *ws.Handler
	Session   *session.Manager
	Workspace string   // 用于 skill 列表
	StoreDir  string   // 用于 memory 读写
	CA        *auth.CA // 用于设备证书签发
}

func NewRouter(deps Dependencies, authToken string) http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"` + deps.Version + `"}`))
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
	})

	if deps.FS != nil {
		r.Handle("/*", spaHandler(deps.FS))
	}

	return r
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
