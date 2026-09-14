// Package controller (dashboard) = HTTP handler statistik admin.
package controller

import (
	"net/http"

	"undangan/services/dashboard/service"
	"undangan/kernel/httpx"
)

type Controller struct{ svc service.Service }

func New(svc service.Service) *Controller { return &Controller{svc: svc} }

func (c *Controller) Register(rt *httpx.Router) {
	rt.Admin("GET /api/v1/admin/stats", c.stats)
}

func (c *Controller) stats(w http.ResponseWriter, r *http.Request) error {
	s, err := c.svc.Stats(r.Context())
	if err != nil {
		return err
	}
	httpx.OK(w, s)
	return nil
}
