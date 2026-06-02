package gateway

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jaycrl/mytool/internal/store"
)

func memoryListHandler(storeDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memories, err := store.ListMemory(storeDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(memories)
	}
}

func memoryUpdateHandler(storeDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/memory/")
		if name == "" {
			http.Error(w, "name required", http.StatusBadRequest)
			return
		}
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := store.SaveMemory(storeDir, name, req.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
