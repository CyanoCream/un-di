package controller

import (
	"net/http"
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/service"
	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
)

type WishController struct {
	svc     service.WishService
	limiter *httpx.RateLimiter
}

func NewWishController(svc service.WishService) *WishController {
	return &WishController{svc: svc, limiter: httpx.NewRateLimiter(5, 10*time.Minute)}
}

func (c *WishController) Register(rt *httpx.Router) {
	rt.Public("GET /api/v1/public/invitations/{id}/wishes", c.publicList)
	rt.Public("POST /api/v1/public/invitations/{id}/wishes", c.publicCreate)

	rt.Auth("GET /api/v1/invitations/{id}/wishes", c.list)
	rt.Auth("PATCH /api/v1/invitations/{id}/wishes/{wid}", c.setHidden)
	rt.Auth("DELETE /api/v1/invitations/{id}/wishes/{wid}", c.delete)
}

func (c *WishController) publicList(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	items, stats, total, err := c.svc.PublicList(r.Context(), r.PathValue("id"), p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{
		"items": items, "total": total, "page": p.Page, "per_page": p.PerPage,
		"stats": map[string]int{"hadir": stats.Hadir, "tidak": stats.Tidak, "ragu": stats.Ragu},
	})
	return nil
}

func (c *WishController) publicCreate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if !c.limiter.Allow(httpx.ClientIP(r) + "|" + id) {
		return apperror.TooManyRequests()
	}
	var in domain.WishInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	wish, err := c.svc.PublicCreate(r.Context(), id, in, httpx.ClientIP(r))
	if err != nil {
		return err
	}
	httpx.Created(w, wish)
	return nil
}

func (c *WishController) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	items, stats, total, err := c.svc.List(r.Context(), actor(r), r.PathValue("id"), p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	if items == nil {
		items = []domain.Wish{}
	}
	httpx.OK(w, map[string]any{"items": items, "total": total, "page": p.Page, "per_page": p.PerPage, "stats": stats})
	return nil
}

func (c *WishController) setHidden(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IsHidden bool `json:"is_hidden"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	wish, err := c.svc.SetHidden(r.Context(), actor(r), r.PathValue("id"), r.PathValue("wid"), in.IsHidden)
	if err != nil {
		return err
	}
	httpx.OK(w, wish)
	return nil
}

func (c *WishController) delete(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.Delete(r.Context(), actor(r), r.PathValue("id"), r.PathValue("wid")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}
