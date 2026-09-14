// Package controller (theme) = API katalog tema. Render & aset tema ada di service invitation.
package controller

import (
	"net/http"

	"undangan/kernel/authctx"
	"undangan/kernel/httpx"
	"undangan/services/theme/domain"
	"undangan/services/theme/service"
)

type Controller struct {
	svc service.Service
}

func New(svc service.Service) *Controller {
	return &Controller{svc: svc}
}

func (c *Controller) Register(rt *httpx.Router) {
	rt.Auth("GET /api/v1/themes", c.listActive)
	rt.Admin("GET /api/v1/admin/themes", c.listAll)
	rt.Admin("PATCH /api/v1/admin/themes/{slug}", c.update)
}

func (c *Controller) listActive(w http.ResponseWriter, r *http.Request) error {
	items, err := c.svc.List(r.Context(), true)
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *Controller) listAll(w http.ResponseWriter, r *http.Request) error {
	items, err := c.svc.List(r.Context(), false)
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *Controller) update(w http.ResponseWriter, r *http.Request) error {
	var in domain.UpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	t, err := c.svc.Update(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("slug"), in)
	if err != nil {
		return err
	}
	httpx.OK(w, t)
	return nil
}
