package web

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// frontendHandler serves the React bundle built in idp/frontend (ADR-017)
// under /ui/. Paths without a matching file return index.html so browser
// routes survive a refresh. When the bundle is missing only /ui/ is affected.
func frontendHandler(dir string) http.Handler {
	files := http.StripPrefix("/ui/", http.FileServer(http.Dir(dir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		index := filepath.Join(dir, "index.html")
		if _, err := os.Stat(index); err != nil {
			http.Error(w, "The React UI is not built. Run `npm ci && npm run build` in idp/frontend, or set IDP_FRONTEND_DIR.", http.StatusNotFound)
			return
		}
		rel := strings.TrimPrefix(path.Clean(r.URL.Path), "/ui")
		rel = strings.TrimPrefix(rel, "/")
		if rel != "" {
			if info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
				if strings.HasPrefix(rel, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				files.ServeHTTP(w, r)
				return
			}
			if strings.HasPrefix(rel, "assets/") {
				http.NotFound(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, index)
	})
}
