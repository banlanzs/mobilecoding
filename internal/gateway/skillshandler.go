package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/jaycrl/mytool/internal/store"
)

func skillsHandler(workspace string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		skills, err := store.ListSkills(workspace)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(skills)
	}
}
