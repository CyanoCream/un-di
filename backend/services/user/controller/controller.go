// Package controller (user) = HTTP handler profil & manajemen user.
package controller

import (
	"context"
	"net/http"

	"undangan/services/user/domain"
	"undangan/services/user/service"
	"undangan/kernel/httpx"
	"undangan/kernel/authctx"
)

// Port lintas modul untuk halaman detail user di admin (diisi saat wiring di server).
type SubscriptionFinder interface {
	CurrentForUser(ctx context.Context, userID string) (any, error)
}

type SubscriptionGranter interface {
	GrantByAdmin(ctx context.Context, actor authctx.Principal, userID, planID string, days int) (any, error)
}

type InvitationLister interface {
	ListForUser(ctx context.Context, userID string) (any, error)
}

type Controller struct {
	svc         service.Service
	subs        SubscriptionFinder
	granter     SubscriptionGranter
	invitations InvitationLister
}

func New(svc service.Service, subs SubscriptionFinder, granter SubscriptionGranter, invitations InvitationLister) *Controller {
	return &Controller{svc: svc, subs: subs, granter: granter, invitations: invitations}
}

func (c *Controller) Register(rt *httpx.Router) {
	rt.Customer("GET /api/v1/me/profile", c.getProfile)
	rt.Customer("PATCH /api/v1/me/profile", c.updateProfile)
	rt.Auth("POST /api/v1/me/password", c.changePassword)

	rt.Admin("GET /api/v1/admin/users", c.list)
	rt.Admin("POST /api/v1/admin/users", c.create)
	rt.Admin("GET /api/v1/admin/users/{id}", c.detail)
	rt.Admin("PATCH /api/v1/admin/users/{id}", c.adminUpdate)
	rt.Admin("POST /api/v1/admin/users/{id}/reset-password", c.resetPassword)
}

func (c *Controller) getProfile(w http.ResponseWriter, r *http.Request) error {
	u, err := c.svc.Get(r.Context(), authctx.MustFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"user": u})
	return nil
}

func (c *Controller) updateProfile(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Name, Phone string }
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	u, err := c.svc.UpdateProfile(r.Context(), authctx.MustFrom(r.Context()).UserID, in.Name, in.Phone)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"user": u})
	return nil
}

func (c *Controller) changePassword(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if err := c.svc.ChangePassword(r.Context(), authctx.MustFrom(r.Context()).UserID, in.CurrentPassword, in.NewPassword); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (c *Controller) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.svc.List(r.Context(), domain.ListFilter{Query: q.Get("q"), Role: q.Get("role"), Limit: p.PerPage, Offset: p.Offset()})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}

func (c *Controller) create(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		service.RegisterInput
		PlanID string `json:"plan_id"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	actor := authctx.MustFrom(r.Context())
	u, err := c.svc.CreateCustomer(r.Context(), actor, in.RegisterInput)
	if err != nil {
		return err
	}
	if in.PlanID != "" {
		if _, err := c.granter.GrantByAdmin(r.Context(), actor, u.ID, in.PlanID, 0); err != nil {
			return err
		}
	}
	httpx.Created(w, map[string]any{"user": u})
	return nil
}

func (c *Controller) detail(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	u, err := c.svc.Get(r.Context(), id)
	if err != nil {
		return err
	}
	sub, err := c.subs.CurrentForUser(r.Context(), id)
	if err != nil {
		return err
	}
	invs, err := c.invitations.ListForUser(r.Context(), id)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"user": u, "subscription": sub, "invitations": invs})
	return nil
}

func (c *Controller) adminUpdate(w http.ResponseWriter, r *http.Request) error {
	var in service.AdminUpdateInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	u, err := c.svc.AdminUpdate(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"), in)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"user": u})
	return nil
}

func (c *Controller) resetPassword(w http.ResponseWriter, r *http.Request) error {
	pw, err := c.svc.ResetPassword(r.Context(), authctx.MustFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]string{"password": pw})
	return nil
}
