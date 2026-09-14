package server

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SPA melayani hasil build Vite (dist/) untuk satu host: file statis apa adanya,
// rute lain (mis. /undangan/123) jatuh ke index.html supaya vue-router yang menangani.
func SPA(dir, name string) http.Handler {
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if _, err := os.Stat(index); err != nil {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			http.Error(w, name+" belum di-build. Jalankan: npm run build", http.StatusServiceUnavailable)
			return
		}
		clean := path.Clean("/" + r.URL.Path)
		file := filepath.Join(dir, filepath.FromSlash(clean))
		if st, err := os.Stat(file); err == nil && !st.IsDir() && clean != "/index.html" {
			if strings.HasPrefix(clean, "/assets/") {
				// Nama file Vite berisi hash konten → aman di-cache selamanya.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, file)
			return
		}
		if strings.HasPrefix(clean, "/assets/") {
			http.NotFound(w, r) // aset hilang jangan dibalas index.html
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		http.ServeFile(w, r, index)
	})
}
