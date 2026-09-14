// Package service (landing) menyusun LandingData dari port tema, paket, dan kontak,
// dengan cache memori singkat supaya halaman publik tetap cepat.
package service

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"undangan/services/landing/domain"
)

// Alias port supaya pemanggil cukup mengimpor package service.
type (
	ThemeLister     = domain.ThemeLister
	PlanLister      = domain.PlanLister
	ContactProvider = domain.ContactProvider
)

type Config struct {
	AppName    string // brand, mis. "Undangin"
	PortalURL  string // portal customer, mis. https://app.undangin.id
	BaseDomain string // mis. undangin.id; "localhost" saat dev
}

type Service interface {
	// Landing mengembalikan data halaman (salinan dari cache ≤ TTL). Kegagalan sebagian port
	// tidak menggagalkan halaman: section terkait kosong dan Degraded=true.
	Landing(ctx context.Context) (*domain.LandingData, error)
}

const (
	// CacheTTL = umur cache data lengkap.
	CacheTTL = 60 * time.Second
	// degradedTTL = umur cache bila ada port yang gagal (dicoba lagi lebih cepat).
	degradedTTL = 5 * time.Second
	portTimeout = 3 * time.Second
)

type service struct {
	themes  ThemeLister
	plans   PlanLister
	contact ContactProvider
	cfg     Config
	now     func() time.Time
	log     *slog.Logger

	mu        sync.Mutex
	cached    *domain.LandingData
	expiresAt time.Time
}

// New: contact boleh nil (section WhatsApp disembunyikan).
func New(themes ThemeLister, plans PlanLister, contact ContactProvider, cfg Config) Service {
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	if cfg.AppName == "" {
		cfg.AppName = "Undangin"
	}
	cfg.PortalURL = strings.TrimRight(strings.TrimSpace(cfg.PortalURL), "/")
	cfg.BaseDomain = strings.TrimSpace(cfg.BaseDomain)
	return &service{themes: themes, plans: plans, contact: contact, cfg: cfg, now: time.Now, log: slog.Default()}
}

func (s *service) Landing(ctx context.Context) (*domain.LandingData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	if s.cached != nil && now.Before(s.expiresAt) {
		return s.cached, nil
	}
	d := s.build(ctx, now)
	ttl := CacheTTL
	if d.Degraded {
		ttl = degradedTTL
		// Data lama yang lengkap lebih baik daripada halaman bolong.
		if s.cached != nil && !s.cached.Degraded {
			s.expiresAt = now.Add(ttl)
			return s.cached, nil
		}
	}
	s.cached, s.expiresAt = d, now.Add(ttl)
	return d, nil
}

func (s *service) build(ctx context.Context, now time.Time) *domain.LandingData {
	ctx, cancel := context.WithTimeout(ctx, portTimeout)
	defer cancel()

	d := &domain.LandingData{
		Brand:       s.cfg.AppName,
		SiteURL:     SiteURL(s.cfg.BaseDomain),
		BaseDomain:  s.cfg.BaseDomain,
		PortalURL:   s.cfg.PortalURL,
		RegisterURL: s.cfg.PortalURL + "/daftar",
		LoginURL:    s.cfg.PortalURL + "/masuk",
		Year:        now.Year(),
		GeneratedAt: now,
	}

	var themes []domain.Theme
	if s.themes != nil {
		t, err := s.themes.ListThemes(ctx)
		if err != nil {
			s.log.Warn("landing: list themes", "err", err)
			d.Degraded = true
		}
		themes = t
	}
	var plans []domain.Plan
	if s.plans != nil {
		p, err := s.plans.ListPlans(ctx)
		if err != nil {
			s.log.Warn("landing: list plans", "err", err)
			d.Degraded = true
		}
		plans = p
	}
	if s.contact != nil {
		c, err := s.contact.Contact(ctx)
		if err != nil {
			s.log.Warn("landing: contact", "err", err)
			d.Degraded = true
		} else {
			msg := "Halo admin " + d.Brand + ", saya ingin dibuatkan undangan digital. Mohon info langkah selanjutnya, terima kasih."
			d.WhatsAppURL = domain.WhatsAppLink(c.AdminWhatsApp, msg)
			if d.WhatsAppURL != "" {
				d.WhatsAppDisplay = domain.WhatsAppDisplay(c.AdminWhatsApp)
			}
			d.MerchantName = strings.TrimSpace(c.MerchantName)
		}
	}

	d.Themes, d.Categories, d.HeroTheme = domain.BuildThemes(themes)
	d.Plans = domain.BuildPlans(plans)
	d.Facts = domain.BuildFacts(d.Themes, plans)
	d.Features = domain.BuildFeatures(d.Facts, s.cfg.BaseDomain)
	d.Steps = domain.BuildSteps()
	d.FAQ = domain.BuildFAQ(d.Facts, plans, s.cfg.BaseDomain, d.MerchantName, d.WhatsAppURL != "")
	return d
}

// SiteURL: "undangin.id" → "https://undangin.id"; localhost/kosong → "http://localhost".
func SiteURL(base string) string {
	base = strings.TrimSpace(base)
	if base == "" || base == "localhost" || strings.HasSuffix(base, ".localhost") || strings.HasPrefix(base, "127.") {
		if base == "" {
			base = "localhost"
		}
		return "http://" + base
	}
	return "https://" + base
}
