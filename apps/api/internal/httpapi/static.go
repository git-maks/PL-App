package httpapi

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// staticHandler serves the prerendered SvelteKit build: exact files, prerendered pages
// ("/hero" → "hero.html") and the SPA fallback (200.html) for dynamic routes like /q/… and /t/….
func (s *Server) staticHandler() http.Handler {
	files := http.FileServerFS(s.static)
	exists := func(name string) bool {
		st, err := fs.Stat(s.static, name)
		return err == nil && !st.IsDir()
	}
	serveFile := func(w http.ResponseWriter, r *http.Request, name string) {
		b, err := fs.ReadFile(s.static, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(b)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		switch {
		case p == "" || p == ".":
			serveFile(w, r, "index.html")
			return
		case strings.HasPrefix(p, "_app/immutable/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case p == "service-worker.js" || strings.HasSuffix(p, ".webmanifest"):
			w.Header().Set("Cache-Control", "no-cache")
		default:
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		if exists(p) {
			files.ServeHTTP(w, r)
			return
		}
		if exists(p + ".html") {
			serveFile(w, r, p+".html")
			return
		}
		if exists(path.Join(p, "index.html")) {
			serveFile(w, r, path.Join(p, "index.html"))
			return
		}
		// Missing assets are 404s, but QR deep links (/q/1.hex.window.mac, /q/s.hex.mac) contain
		// dots too and must reach the app.
		if path.Ext(p) != "" && !strings.HasPrefix(p, "q/") && !strings.HasPrefix(p, "t/") {
			http.NotFound(w, r)
			return
		}
		serveFile(w, r, "200.html")
	})
}
