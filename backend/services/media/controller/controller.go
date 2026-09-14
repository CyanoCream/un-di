// Package controller (media) = HTTP handler upload, pustaka musik, dan penyajian file /uploads.
package controller

import (
	"io/fs"
	"net/http"

	"undangan/services/media/service"
	"undangan/kernel/httpx"
	"undangan/kernel/storage"
	"undangan/kernel/authctx"
)

type Controller struct {
	svc   service.Service
	files storage.FileStorage
}

func New(svc service.Service, files storage.FileStorage) *Controller {
	return &Controller{svc: svc, files: files}
}

func (c *Controller) Register(rt *httpx.Router) {
	rt.Auth("POST /api/v1/uploads", c.upload)
	rt.Auth("GET /api/v1/music", c.listMusic)

	rt.Admin("GET /api/v1/admin/music", c.listMusic)
	rt.Admin("POST /api/v1/admin/music", c.createMusic)
	rt.Admin("DELETE /api/v1/admin/music/{id}", c.deleteMusic)
}

func (c *Controller) upload(w http.ResponseWriter, r *http.Request) error {
	// Batas terbesar dulu (audio), baru kind diketahui dari form/query.
	if err := httpx.ParseMultipart(w, r, storage.AudioRule.MaxBytes); err != nil {
		return err
	}
	kind, rule, err := service.RuleFor(r.FormValue("kind"))
	if err != nil {
		return err
	}
	data, _, err := httpx.FormFile(r, rule.MaxBytes)
	if err != nil {
		return err
	}
	url, err := c.svc.Upload(r.Context(), authctx.MustFrom(r.Context()), kind, data)
	if err != nil {
		return err
	}
	httpx.Created(w, map[string]string{"url": url})
	return nil
}

func (c *Controller) listMusic(w http.ResponseWriter, r *http.Request) error {
	items, err := c.svc.ListMusic(r.Context())
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *Controller) createMusic(w http.ResponseWriter, r *http.Request) error {
	if err := httpx.ParseMultipart(w, r, storage.AudioRule.MaxBytes); err != nil {
		return err
	}
	data, _, err := httpx.FormFile(r, storage.AudioRule.MaxBytes)
	if err != nil {
		return err
	}
	t, err := c.svc.CreateMusic(r.Context(), authctx.MustFrom(r.Context()), r.FormValue("title"), r.FormValue("artist"), data)
	if err != nil {
		return err
	}
	httpx.Created(w, t)
	return nil
}

func (c *Controller) deleteMusic(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.DeleteMusic(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

// ServeUploads menyajikan file publik; pasang dengan pola "GET /uploads/{path...}".
// Nama file acak & tidak pernah ditimpa → aman di-cache permanen.
func (c *Controller) ServeUploads(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	if name == "" || !fs.ValidPath(name) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFileFS(w, r, c.files.PublicFS(), name)
}
