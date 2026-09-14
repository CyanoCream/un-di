package controller

import (
	"context"
	"net/http"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/service"
	"undangan/services/invitation/renderer"
	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
	"undangan/kernel/authctx"
)

type InvitationController struct {
	svc      service.InvitationService
	themes   domain.ThemeChecker
	renderer *renderer.Renderer
	url      domain.URLBuilder
	domains  domain.DomainConfig
}

func NewInvitationController(svc service.InvitationService, themes domain.ThemeChecker, r *renderer.Renderer, url domain.URLBuilder, domains domain.DomainConfig) *InvitationController {
	return &InvitationController{svc: svc, themes: themes, renderer: r, url: url, domains: domains}
}

func (c *InvitationController) Register(rt *httpx.Router) {
	rt.Public("GET /api/v1/subdomains/check", c.checkSubdomain)

	rt.Auth("GET /api/v1/invitations", c.list)
	rt.Auth("POST /api/v1/invitations", c.create)
	rt.Auth("GET /api/v1/invitations/{id}", c.get)
	rt.Auth("PATCH /api/v1/invitations/{id}", c.update)
	rt.Admin("DELETE /api/v1/invitations/{id}", c.delete)
	rt.Auth("PUT /api/v1/invitations/{id}/theme", c.setTheme)
	rt.Auth("PUT /api/v1/invitations/{id}/subdomain", c.setSubdomain)
	rt.Auth("POST /api/v1/invitations/{id}/publish", c.publish)
	rt.Auth("POST /api/v1/invitations/{id}/unpublish", c.unpublish)
	rt.Auth("GET /api/v1/invitations/{id}/preview", c.preview)
	rt.Auth("PUT /api/v1/invitations/{id}/settings", c.updateSettings)
}

// ListForUser memenuhi port user/controller.InvitationLister (halaman detail user di admin).
func (c *InvitationController) ListForUser(ctx context.Context, userID string) (any, error) {
	items, err := c.svc.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toSummaries(items, c.url), nil
}

func actor(r *http.Request) authctx.Principal { return authctx.MustFrom(r.Context()) }

func (c *InvitationController) respond(w http.ResponseWriter, status int, d *service.Details, err error) error {
	if err != nil {
		return err
	}
	httpx.JSON(w, status, toDetail(d, c.url, c.domains))
	return nil
}

func (c *InvitationController) checkSubdomain(w http.ResponseWriter, r *http.Request) error {
	name := r.URL.Query().Get("name")
	ok, reason, err := c.svc.CheckSubdomain(r.Context(), name)
	if err != nil {
		return err
	}
	body := map[string]any{"name": name, "available": ok}
	if reason != "" {
		body["reason"] = reason
	}
	httpx.OK(w, body)
	return nil
}

func (c *InvitationController) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.svc.List(r.Context(), actor(r), domain.ListFilter{
		UserID: q.Get("user_id"), Query: q.Get("q"), Status: q.Get("status"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(toSummaries(items, c.url), total, p))
	return nil
}

func (c *InvitationController) create(w http.ResponseWriter, r *http.Request) error {
	var in service.CreateInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	d, err := c.svc.Create(r.Context(), actor(r), in)
	return c.respond(w, http.StatusCreated, d, err)
}

func (c *InvitationController) get(w http.ResponseWriter, r *http.Request) error {
	d, err := c.svc.Get(r.Context(), actor(r), r.PathValue("id"))
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) update(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Content *domain.Content `json:"content"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if in.Content == nil {
		return apperror.Validation(map[string]string{"content": "Konten wajib dikirim"})
	}
	d, err := c.svc.UpdateContent(r.Context(), actor(r), r.PathValue("id"), *in.Content)
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) delete(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.Delete(r.Context(), actor(r), r.PathValue("id")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (c *InvitationController) setTheme(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Theme string }
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	d, err := c.svc.SetTheme(r.Context(), actor(r), r.PathValue("id"), in.Theme)
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) setSubdomain(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Name string }
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	d, err := c.svc.SetSubdomain(r.Context(), actor(r), r.PathValue("id"), in.Name)
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) updateSettings(w http.ResponseWriter, r *http.Request) error {
	var in domain.SettingsInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	d, err := c.svc.UpdateSettings(r.Context(), actor(r), r.PathValue("id"), in)
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) publish(w http.ResponseWriter, r *http.Request) error {
	d, err := c.svc.Publish(r.Context(), actor(r), r.PathValue("id"))
	return c.respond(w, http.StatusOK, d, err)
}

func (c *InvitationController) unpublish(w http.ResponseWriter, r *http.Request) error {
	d, err := c.svc.Unpublish(r.Context(), actor(r), r.PathValue("id"))
	return c.respond(w, http.StatusOK, d, err)
}

// preview merender undangan dengan data milik sendiri (untuk iframe editor). ?theme= untuk mencoba tema lain.
func (c *InvitationController) preview(w http.ResponseWriter, r *http.Request) error {
	p := actor(r)
	inv, err := c.svc.Load(r.Context(), p, r.PathValue("id"))
	if err != nil {
		return err
	}
	theme := r.URL.Query().Get("theme")
	if theme == "" && inv.Theme != nil {
		theme = *inv.Theme
	}
	if theme == "" {
		return apperror.Unprocessable("theme_required", "Pilih tema untuk melihat pratinjau")
	}
	ok, err := c.themes.IsSelectable(r.Context(), theme, p.IsAdmin())
	if err != nil {
		return err
	}
	if !ok {
		return apperror.NotFound("Tema tidak ditemukan")
	}
	view := &renderer.Invitation{ID: inv.ID, Theme: theme}
	guestName := renderer.GuestNameFromQuery(r.URL.Query().Get("to"))
	if guestName == "" {
		guestName = "Nama Tamu"
	}
	// Pratinjau menampilkan tiket contoh bila check-in aktif.
	var guest *renderer.GuestView
	if inv.CheckinEnabled {
		guest = &renderer.GuestView{Name: guestName, Pax: 2, Code: "CONTOH", Slug: "nama-tamu"}
	}
	data := renderer.BuildPageData(view, inv.Content, guest, guestName, true, "").WithAccess(inv.AccessMode, inv.CheckinEnabled)
	RenderHTML(w, c.renderer, http.StatusOK, theme, data)
	return nil
}
