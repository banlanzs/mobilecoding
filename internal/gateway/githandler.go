package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/banlanzs/mobilecoding/internal/files"
)

// gitStatusHandler 返回 git status 文件列表。
// GET /api/v1/git/status?cwd=/path/to/repo
func gitStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd := r.URL.Query().Get("cwd")

		status, err := files.GetGitStatus(cwd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"files": status,
		})
	}
}

// gitDiffHandler 返回指定文件的 diff 内容。
// GET /api/v1/git/diff?cwd=/path/to/repo&file=path/to/file
func gitDiffHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd := r.URL.Query().Get("cwd")
		file := r.URL.Query().Get("file")

		diff, err := files.GetGitDiff(cwd, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"diff": diff,
		})
	}
}

// gitDiffSummaryHandler 返回 git diff 的统计摘要。
// GET /api/v1/git/diff-summary?cwd=/path/to/repo
func gitDiffSummaryHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd := r.URL.Query().Get("cwd")

		summary, err := files.GetGitDiffSummary(cwd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summary)
	}
}
