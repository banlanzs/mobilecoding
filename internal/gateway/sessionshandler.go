package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/banlanzs/mobilecoding/internal/session"
)

// sessionsListHandler 返回所有会话列表。
// GET /api/v1/sessions
func sessionsListHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mgr == nil {
			http.Error(w, "session manager not available", http.StatusServiceUnavailable)
			return
		}

		sessions, err := mgr.ListSessions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessions,
		})
	}
}

// sessionsCreateHandler 创建新会话。
// POST /api/v1/sessions
// Body: { "agent": "claude", "model": "claude-sonnet-4-6", "cwd": "/path/to/workspace" }
func sessionsCreateHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mgr == nil {
			http.Error(w, "session manager not available", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			Agent string `json:"agent"`
			Model string `json:"model"`
			CWD   string `json:"cwd"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Agent == "" {
			http.Error(w, "agent is required", http.StatusBadRequest)
			return
		}

		// 注意：实际的会话启动通过 WebSocket session.start RPC 完成
		// 这个 API 仅用于预创建会话元数据（供前端展示占位符）
		// 当前实现暂不支持预创建，返回错误提示
		http.Error(w, "session creation via REST API not yet implemented - use WebSocket session.start", http.StatusNotImplemented)
	}
}

// sessionsGetHandler 获取单个会话详情。
// GET /api/v1/sessions/:id
func sessionsGetHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mgr == nil {
			http.Error(w, "session manager not available", http.StatusServiceUnavailable)
			return
		}

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}

		meta, err := mgr.GetSession(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(meta)
	}
}

// sessionsDeleteHandler 删除会话。
// DELETE /api/v1/sessions/:id
func sessionsDeleteHandler(mgr *session.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if mgr == nil {
			http.Error(w, "session manager not available", http.StatusServiceUnavailable)
			return
		}

		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}

		if err := mgr.DeleteSession(id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}
}
