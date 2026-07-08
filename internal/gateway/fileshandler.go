package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/banlanzs/mobilecoding/internal/files"
)

// filesTreeHandler 返回工作区目录树。
// GET /api/v1/files/tree?cwd=/path&depth=3&limit=200
func filesTreeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd := r.URL.Query().Get("cwd")
		if cwd == "" {
			cwd = "."
		}
		depth := 3
		if v := r.URL.Query().Get("depth"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 6 {
				depth = n
			}
		}
		limit := 200
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
				limit = n
			}
		}

		tree, err := files.ListTree(cwd, depth, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tree": tree,
		})
	}
}

// filesReadHandler 读取工作区内指定文件内容。
// GET /api/v1/files/read?cwd=/path&file=src/main.go&maxSize=204800
func filesReadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd := r.URL.Query().Get("cwd")
		if cwd == "" {
			cwd = "."
		}
		file := r.URL.Query().Get("file")
		if file == "" {
			http.Error(w, "file required", http.StatusBadRequest)
			return
		}
		maxSize := 200 * 1024
		if v := r.URL.Query().Get("maxSize"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5*1024*1024 {
				maxSize = n
			}
		}

		content, err := files.ReadFile(cwd, file, maxSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(content)
	}
}
