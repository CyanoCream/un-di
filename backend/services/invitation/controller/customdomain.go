package controller

import (
	"net/http"

	"undangan/services/invitation/service"
	"undangan/kernel/httpx"
)

type CustomDomainController struct {
	svc  service.CustomDomainService
	view *InvitationController // pembentuk DTO detail yang sama dengan endpoint undangan
}

func NewCustomDomainController(svc service.CustomDomainService, view *InvitationController) *CustomDomainController {
	return &CustomDomainController{svc: svc, view: view}
}

func (c *CustomDomainController) Register(rt *httpx.Router) {
	rt.Auth("PUT /api/v1/invitations/{id}/custom-domain", c.set)
	rt.Auth("POST /api/v1/invitations/{id}/custom-domain/verify", c.verify)
	rt.Auth("DELETE /api/v1/invitations/{id}/custom-domain", c.remove)
}

func (c *CustomDomainController) set(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Hostname string }
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	d, err := c.svc.Set(r.Context(), actor(r), r.PathValue("id"), in.Hostname)
	return c.view.respond(w, http.StatusOK, d, err)
}

func (c *CustomDomainController) verify(w http.ResponseWriter, r *http.Request) error {
	d, err := c.svc.Verify(r.Context(), actor(r), r.PathValue("id"))
	return c.view.respond(w, http.StatusOK, d, err)
}

func (c *CustomDomainController) remove(w http.ResponseWriter, r *http.Request) error {
	d, err := c.svc.Remove(r.Context(), actor(r), r.PathValue("id"))
	return c.view.respond(w, http.StatusOK, d, err)
}
