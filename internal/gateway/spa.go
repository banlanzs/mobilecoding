package gateway

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func spaHandler(fsys fs.FS) http.HandlerFunc {
	fileServer := http.FileServer(http.FS(fsys))
	return func(w http.ResponseWriter, r *http.Request) {
		upath := r.URL.Path
		if upath == "" || upath == "/" {
			upath = "/index.html"
		}
		clean := path.Clean(upath)
		if strings.HasPrefix(clean, "..") || strings.Contains(clean, "/../") {
			http.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(fsys, strings.TrimPrefix(clean, "/")); err == nil {
			if strings.HasSuffix(clean, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		if strings.HasSuffix(clean, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		}
		fileServer.ServeHTTP(w, r2)
	}
}
