// Package gateway 提供 mytool HTTP 入口：healthz/version/SPA + REST 占位 + WS 升级。
package gateway

import (
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
	Workspace string // 用于 skill 列表
	StoreDir  string // 用于 memory 读写
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
	})

	if deps.FS != nil {
		r.Handle("/*", spaHandler(deps.FS))
	}

	return r
}
