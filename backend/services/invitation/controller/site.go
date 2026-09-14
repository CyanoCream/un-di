package controller

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/service"
	"undangan/services/invitation/renderer"
	"undangan/kernel/apperror"
	"undangan/kernel/tenant"
)

// SiteController melayani halaman undangan untuk tamu berdasarkan Host (subdomain / custom domain).
//
//	GET /                 undangan umum (mode public) atau halaman gerbang (mode guest_only)
//	GET /?kode=K7P2QX     buka dengan kode undangan → redirect ke /<slug>
//	GET /<slug>           link personal; slug harus terdaftar di daftar tamu
//	GET /<KODE>           link personal via kode (QR/passcode) → redirect ke /<slug>
type SiteController struct {
	svc        service.SiteService
	renderer   *renderer.Renderer
	baseDomain string
}

func NewSiteController(svc service.SiteService, r *renderer.Renderer, baseDomain string) *SiteController {
	return &SiteController{svc: svc, renderer: r, baseDomain: baseDomain}
}

func (c *SiteController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	inv, err := c.svc.Resolve(r.Context(), tenant.Parse(r.Host, c.baseDomain))
	if err != nil || inv.Theme == nil {
		c.notFound(w, err)
		return
	}

	scheme := "https"
	if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") != "https" {
		scheme = "http"
	}
	origin := scheme + "://" + r.Host
	// Halaman berisi nama tamu → jangan di-cache proxy bersama; jangan diindeks mesin pencari.
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("X-Robots-Tag", "noindex")

	path := strings.Trim(r.URL.Path, "/")
	q := r.URL.Query()

	// Form "kode undangan" di halaman gerbang.
	if path == "" && q.Get("kode") != "" {
		if code := domain.ExtractGuestCode(q.Get("kode")); code != "" {
			if g, err := c.svc.OpenGuest(r.Context(), inv.ID, code); err == nil {
				http.Redirect(w, r, "/"+url.PathEscape(g.Slug), http.StatusSeeOther)
				return
			}
		}
		c.gate(w, http.StatusOK, inv, origin, "guest_only", "", "Kode undangan tidak valid. Periksa kembali kode Anda.")
		return
	}

	var guest *renderer.GuestView
	switch {
	case path == "":
		if inv.AccessMode == domain.AccessGuestOnly {
			c.gate(w, http.StatusOK, inv, origin, "guest_only", "", "")
			return
		}
	case strings.Contains(path, "/"):
		c.notFound(w, apperror.ErrNotFound)
		return
	default:
		g, err := c.svc.OpenGuest(r.Context(), inv.ID, path)
		if err != nil {
			var ae *apperror.Error
			if errors.As(err, &ae) && ae.Status == http.StatusNotFound {
				// Nama tidak ada di daftar tamu → bukan undangan.
				c.gate(w, http.StatusNotFound, inv, origin, "not_registered", displayName(path), "")
				return
			}
			c.notFound(w, err)
			return
		}
		// Kode di path (dari QR/passcode) → arahkan ke link nama yang rapi.
		if !strings.EqualFold(path, g.Slug) {
			target := "/" + url.PathEscape(g.Slug)
			if code := q.Get("c"); code != "" {
				target += "?c=" + url.QueryEscape(code)
			}
			http.Redirect(w, r, target, http.StatusFound)
			return
		}
		guest = &renderer.GuestView{Name: g.Name, Pax: g.Pax, Code: g.Code, Slug: g.Slug, Link: origin + "/" + g.Slug}
	}

	guestName := ""
	if guest == nil && inv.AccessMode != domain.AccessGuestOnly {
		guestName = renderer.GuestNameFromQuery(q.Get("to")) // mode publik: ?to= tetap didukung (tidak divalidasi)
	}
	data := renderer.BuildPageData(&renderer.Invitation{ID: inv.ID, Theme: *inv.Theme}, inv.Content, guest, guestName, false, origin+"/").
		WithAccess(inv.AccessMode, inv.CheckinEnabled)
	if data.OGImage != "" && strings.HasPrefix(data.OGImage, "/") {
		data.OGImage = origin + data.OGImage // OG image wajib URL absolut untuk preview WhatsApp
	}
	RenderHTML(w, c.renderer, http.StatusOK, *inv.Theme, data)
}

func (c *SiteController) gate(w http.ResponseWriter, status int, inv *domain.Invitation, origin, reason, attempted, errMsg string) {
	data := renderer.BuildPageData(&renderer.Invitation{ID: inv.ID, Theme: *inv.Theme}, inv.Content, nil, "", false, origin+"/").
		WithAccess(inv.AccessMode, false)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	if err := c.renderer.RenderGate(w, *inv.Theme, renderer.GateData{PageData: data, Reason: reason, Attempted: attempted, Error: errMsg}); err != nil {
		slog.Error("render gate", "theme", *inv.Theme, "err", err)
	}
}

// displayName: "budi-santoso" → "budi santoso" untuk pesan "nama tidak terdaftar".
func displayName(path string) string {
	s, err := url.PathUnescape(path)
	if err != nil {
		s = path
	}
	s = strings.ReplaceAll(s, "-", " ")
	if r := []rune(s); len(r) > 60 {
		s = string(r[:60])
	}
	return s
}

func (c *SiteController) notFound(w http.ResponseWriter, err error) {
	status := http.StatusNotFound
	var ae *apperror.Error
	if err != nil && !errors.Is(err, apperror.ErrNotFound) && !errors.As(err, &ae) {
		status = http.StatusInternalServerError
		slog.Error("site", "err", err)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	msg := "Undangan tidak ditemukan atau sudah tidak aktif."
	if status == http.StatusInternalServerError {
		msg = "Terjadi kesalahan. Silakan coba beberapa saat lagi."
	}
	_, _ = w.Write([]byte(`<!doctype html><html lang="id"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta name="robots" content="noindex"><title>Undangan tidak ditemukan</title>
<body style="margin:0;min-height:100vh;display:grid;place-items:center;font-family:Georgia,serif;background:#faf6f1;color:#3b3029;text-align:center;padding:1rem">
<div><p style="font-size:3rem;margin:0">💌</p><h1 style="font-weight:normal">` + msg + `</h1></div></body></html>`))
}

// TLSAsk = endpoint internal Caddy on_demand_tls: 200 hanya untuk custom domain terverifikasi.
func (c *SiteController) TLSAsk(w http.ResponseWriter, r *http.Request) {
	host := tenant.Parse(r.URL.Query().Get("domain"), c.baseDomain)
	if host.Kind == tenant.KindCustomDomain {
		if ok, err := c.svc.IsVerifiedCustomDomain(r.Context(), host.Name); err == nil && ok {
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	http.Error(w, "not allowed", http.StatusForbidden)
}
