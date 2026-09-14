// Package server menyusun router HTTP dari controller semua modul.
package server

import (
	"log/slog"
	"net/http"
	"strings"

	"undangan/kernel/httpx"
	"undangan/kernel/tenant"
)

// Registrar = controller modul yang mendaftarkan endpoint API.
type Registrar interface {
	Register(rt *httpx.Router)
}

type Deps struct {
	Log        *slog.Logger
	BaseDomain string

	// AuthMiddleware membaca JWT → authctx (modul auth).
	AuthMiddleware func(http.Handler) http.Handler
	API            []Registrar

	ThemePreview httpx.HandlerFunc // GET /_preview/{theme}
	ThemeShared  http.HandlerFunc  // GET /_shared/{path...}
	ThemeAsset   http.HandlerFunc  // GET /_theme/{theme}/{path...}
	Uploads      http.HandlerFunc  // GET /uploads/{path...}
	Site         http.Handler      // halaman undangan by Host
	TLSAsk       http.HandlerFunc  // Caddy on_demand_tls
	Landing      http.Handler      // opsional; host platform
	AdminApp     http.Handler      // admin.<base> → build apps/admin
	PortalApp    http.Handler      // app.<base> → build apps/portal
}

// NewPublic: listener utama di belakang Caddy. Satu binary melayani semua host:
//
//	admin.<base>                build portal admin (SPA)   ·  app.<base>  build portal customer (SPA)
//	/api/v1/...                 JSON API (host apa pun; SPA admin & portal memanggil lewat origin sendiri)
//	/_shared /_theme /uploads   aset statis
//	/_preview/{tema}            katalog tema
//	<slug>.<base> / custom      halaman undangan
func NewPublic(d Deps) http.Handler {
	apiMux := http.NewServeMux()
	rt := httpx.NewRouter(apiMux)
	for _, c := range d.API {
		c.Register(rt)
	}
	apiMux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusNotFound, map[string]any{"error": map[string]string{"code": "not_found", "message": "Endpoint tidak ditemukan"}})
	})
	api := httpx.CSRFGuard(d.AuthMiddleware(apiMux))

	assets := http.NewServeMux()
	assets.HandleFunc("GET /_shared/{path...}", d.ThemeShared)
	assets.HandleFunc("GET /_theme/{theme}/{path...}", d.ThemeAsset)
	assets.HandleFunc("GET /uploads/{path...}", d.Uploads)
	assets.HandleFunc("GET /_preview/{theme}", httpx.Handle(d.ThemePreview))
	assets.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })

	landing := d.Landing
	if landing == nil {
		landing = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html lang="id"><meta charset="utf-8"><title>Undangan Digital</title><h1>Undangan Digital</h1><p>Landing page (segera hadir)</p></html>`))
		})
	}

	return httpx.AccessLog(d.Log, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.HasPrefix(p, "/api/"):
			api.ServeHTTP(w, r)
		case strings.HasPrefix(p, "/_shared/"), strings.HasPrefix(p, "/_theme/"), strings.HasPrefix(p, "/uploads/"),
			strings.HasPrefix(p, "/_preview/"), p == "/healthz":
			assets.ServeHTTP(w, r)
		default:
			switch tenant.Parse(r.Host, d.BaseDomain).Kind {
			case tenant.KindSubdomain, tenant.KindCustomDomain:
				d.Site.ServeHTTP(w, r)
			case tenant.KindAdminApp:
				d.AdminApp.ServeHTTP(w, r)
			case tenant.KindPortalApp:
				d.PortalApp.ServeHTTP(w, r)
			case tenant.KindPlatform:
				landing.ServeHTTP(w, r)
			default:
				http.NotFound(w, r)
			}
		}
	}))
}

// NewInternal: hanya bind ke loopback, tidak di-proxy Caddy.
func NewInternal(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /internal/tls/ask", d.TLSAsk)
	return mux
}
