package controller

import (
	"log/slog"
	"net/http"

	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
	"undangan/kernel/storage"
	"undangan/services/invitation/domain"
	"undangan/services/invitation/renderer"
)

// ThemeSiteController = katalog pratinjau tema (/_preview) dan aset tema (/_shared, /_theme).
type ThemeSiteController struct {
	themes   domain.ThemeChecker
	renderer *renderer.Renderer
	dev      bool
}

func NewThemeSiteController(themes domain.ThemeChecker, r *renderer.Renderer, dev bool) *ThemeSiteController {
	return &ThemeSiteController{themes: themes, renderer: r, dev: dev}
}

// Preview = GET /_preview/{theme}: tema dengan data contoh (tanpa login).
func (c *ThemeSiteController) Preview(w http.ResponseWriter, r *http.Request) error {
	slug := r.PathValue("theme")
	ok, err := c.themes.IsSelectable(r.Context(), slug, true)
	if err != nil {
		return err
	}
	if !ok {
		return apperror.NotFound("Tema tidak ditemukan")
	}
	inv := &renderer.Invitation{ID: "preview", Theme: slug}
	data := renderer.BuildPageData(inv, renderer.DemoContent(), nil, renderer.GuestNameFromQuery(r.URL.Query().Get("to")), true, "")
	RenderHTML(w, c.renderer, http.StatusOK, slug, data)
	return nil
}

func (c *ThemeSiteController) ServeShared(w http.ResponseWriter, r *http.Request) {
	if !c.dev {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeFileFS(w, r, storage.NoDirFS{FS: c.renderer.SharedFS()}, r.PathValue("path"))
}

func (c *ThemeSiteController) ServeAsset(w http.ResponseWriter, r *http.Request) {
	theme := r.PathValue("theme")
	if !renderer.ValidThemeSlug(theme) {
		http.NotFound(w, r)
		return
	}
	if !c.dev {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	http.ServeFileFS(w, r, storage.NoDirFS{FS: c.renderer.AssetsFS(theme)}, r.PathValue("path"))
}

var _ httpx.HandlerFunc = (&ThemeSiteController{}).Preview

// RenderHTML menulis halaman tema dengan header keamanan standar.
func RenderHTML(w http.ResponseWriter, rd *renderer.Renderer, status int, theme string, data renderer.PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	if data.Preview {
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(status)
	if err := rd.Render(w, theme, data); err != nil {
		slog.Error("render theme", "theme", theme, "err", err) // header sudah terkirim; cukup catat
	}
}
