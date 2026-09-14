package controller

import (
	"net/http"

	"undangan/services/audit/service"
	"undangan/kernel/httpx"
)

type Controller struct{ svc service.Service }

func New(svc service.Service) *Controller { return &Controller{svc: svc} }

func (c *Controller) Register(rt *httpx.Router) {
	rt.Admin("GET /api/v1/admin/audit-logs", c.list)
}

func (c *Controller) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	items, total, err := c.svc.List(r.Context(), p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}
