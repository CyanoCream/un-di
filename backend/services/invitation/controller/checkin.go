package controller

import (
	"net/http"
	"strings"
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/service"
	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
	"undangan/kernel/authctx"
)

// CheckinController = API stasiun penerima tamu. Akses: token stasiun (Bearer, dari PIN) atau pemilik/admin login.
type CheckinController struct {
	svc        service.CheckinService
	pinLimiter *httpx.RateLimiter
}

func NewCheckinController(svc service.CheckinService) *CheckinController {
	return &CheckinController{svc: svc, pinLimiter: httpx.NewRateLimiter(8, 15*time.Minute)}
}

func (c *CheckinController) Register(rt *httpx.Router) {
	rt.Public("POST /api/v1/checkin/{id}/session", c.session)
	rt.Public("GET /api/v1/checkin/{id}/summary", c.summary)
	rt.Public("GET /api/v1/checkin/{id}/guests", c.guests)
	rt.Public("POST /api/v1/checkin/{id}/scan", c.scan)
	rt.Public("GET /api/v1/checkin/{id}/recent", c.recent)
}

func (c *CheckinController) authorize(r *http.Request) (*domain.Invitation, string, error) {
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if bearer == r.Header.Get("Authorization") {
		bearer = ""
	}
	var principal *authctx.Principal
	if p, ok := authctx.From(r.Context()); ok {
		principal = &p
	}
	return c.svc.Authorize(r.Context(), r.PathValue("id"), bearer, principal)
}

func invitationBrief(inv *domain.Invitation) map[string]any {
	return map[string]any{"id": inv.ID, "title": inv.Title(), "event_date": inv.EventDate()}
}

func (c *CheckinController) session(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	// Batasi tebakan PIN per IP per undangan.
	if !c.pinLimiter.Allow(httpx.ClientIP(r) + "|" + id) {
		return apperror.TooManyRequests()
	}
	var in struct {
		Pin         string `json:"pin"`
		StationName string `json:"station_name"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	token, exp, inv, err := c.svc.StartSession(r.Context(), id, in.Pin, in.StationName)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"token": token, "expires_at": exp, "invitation": invitationBrief(inv)})
	return nil
}

func (c *CheckinController) summary(w http.ResponseWriter, r *http.Request) error {
	inv, _, err := c.authorize(r)
	if err != nil {
		return err
	}
	sum, err := c.svc.Summary(r.Context(), inv)
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"invitation": invitationBrief(inv), "checkin_enabled": inv.CheckinEnabled, "summary": sum})
	return nil
}

func (c *CheckinController) guests(w http.ResponseWriter, r *http.Request) error {
	inv, _, err := c.authorize(r)
	if err != nil {
		return err
	}
	items, err := c.svc.Search(r.Context(), inv, r.URL.Query().Get("q"))
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}

func (c *CheckinController) scan(w http.ResponseWriter, r *http.Request) error {
	inv, station, err := c.authorize(r)
	if err != nil {
		return err
	}
	var in struct {
		Code string `json:"code"`
		Pax  int    `json:"pax"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	res, err := c.svc.Scan(r.Context(), inv, in.Code, in.Pax, station, httpx.ClientIP(r))
	if err != nil {
		return err
	}
	httpx.OK(w, res)
	return nil
}

func (c *CheckinController) recent(w http.ResponseWriter, r *http.Request) error {
	inv, _, err := c.authorize(r)
	if err != nil {
		return err
	}
	items, err := c.svc.Recent(r.Context(), inv)
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewList(items))
	return nil
}
