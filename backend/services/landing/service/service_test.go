package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"undangan/services/landing/domain"
)

type fakeThemes struct {
	items []domain.Theme
	err   error
	calls int
}

func (f *fakeThemes) ListThemes(context.Context) ([]domain.Theme, error) {
	f.calls++
	return f.items, f.err
}

type fakePlans struct {
	items []domain.Plan
	err   error
	calls int
}

func (f *fakePlans) ListPlans(context.Context) ([]domain.Plan, error) {
	f.calls++
	return f.items, f.err
}

type fakeContact struct {
	c   domain.Contact
	err error
}

func (f *fakeContact) Contact(context.Context) (domain.Contact, error) { return f.c, f.err }

func samplePlans() []domain.Plan {
	return []domain.Plan{
		{ID: "b", Name: "Paket Basic", Price: 100_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 100, AllowCustomDomain: true},
		{ID: "p", Name: "Paket Premium", Price: 175_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 500, AllowCustomDomain: true, AllowCheckin: true},
	}
}

func sampleThemes() []domain.Theme {
	return []domain.Theme{
		{Slug: "jawa-sogan", Name: "Jawa Sogan", Category: "adat", Colors: []string{"#5a3825", "bad", "#c9a15b"}},
		{Slug: "islami-zamrud", Name: "Islami Zamrud", Category: "islami", Thumbnail: "/_theme/islami-zamrud/thumb.webp"},
		{Slug: "bali-kamboja", Name: "Bali Kamboja", Category: "adat"},
		{Slug: "aneh", Name: "Aneh", Category: "tidak-ada"},
	}
}

func newTestService(th *fakeThemes, pl *fakePlans, ct domain.ContactProvider, now *time.Time) *service {
	s := New(th, pl, ct, Config{AppName: "Undangin", PortalURL: "https://app.undangin.id/", BaseDomain: "undangin.id"}).(*service)
	s.now = func() time.Time { return *now }
	return s
}

func TestLandingBuildsData(t *testing.T) {
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	th, pl := &fakeThemes{items: sampleThemes()}, &fakePlans{items: samplePlans()}
	s := newTestService(th, pl, &fakeContact{c: domain.Contact{AdminWhatsApp: "6281234567890", MerchantName: "Undangin"}}, &now)

	d, err := s.Landing(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if d.RegisterURL != "https://app.undangin.id/daftar" || d.LoginURL != "https://app.undangin.id/masuk" {
		t.Fatalf("portal urls: %q %q", d.RegisterURL, d.LoginURL)
	}
	if d.SiteURL != "https://undangin.id" || d.Year != 2026 || d.Degraded {
		t.Fatalf("site=%q year=%d degraded=%v", d.SiteURL, d.Year, d.Degraded)
	}
	if !strings.HasPrefix(d.WhatsAppURL, "https://wa.me/6281234567890?text=") || !strings.Contains(d.WhatsAppURL, "Undangin") {
		t.Fatalf("wa url: %q", d.WhatsAppURL)
	}
	if d.WhatsAppDisplay != "+62 812-3456-7890" {
		t.Fatalf("wa display: %q", d.WhatsAppDisplay)
	}

	// Tema: kategori kanonik, warna invalid dibuang, hero = tema pertama bertumbnail.
	if len(d.Themes) != 4 || d.Themes[0].CategoryLabel != "Adat Nusantara" || len(d.Themes[0].Colors) != 2 {
		t.Fatalf("themes: %+v", d.Themes)
	}
	if d.Themes[0].PreviewURL != "/_preview/jawa-sogan" || d.Themes[3].Category != "" {
		t.Fatalf("theme preview/category: %+v", d.Themes)
	}
	if len(d.Categories) != 2 || d.Categories[0].Key != "adat" || d.Categories[0].Count != 2 || d.Categories[1].Key != "islami" {
		t.Fatalf("categories: %+v", d.Categories)
	}
	if d.HeroTheme == nil || d.HeroTheme.Slug != "islami-zamrud" {
		t.Fatalf("hero: %+v", d.HeroTheme)
	}

	// Paket: harga Rp, highlight paket termurah per tamu.
	if len(d.Plans) != 2 || d.Plans[0].PriceLabel != "Rp100.000" || d.Plans[1].PriceLabel != "Rp175.000" {
		t.Fatalf("plans: %+v", d.Plans)
	}
	if d.Plans[0].Highlight || !d.Plans[1].Highlight || d.Plans[1].Badge == "" {
		t.Fatalf("highlight: %+v", d.Plans)
	}
	if item := findItem(d.Plans[0].Items, "Check-in"); item == nil || item.Included {
		t.Fatalf("basic check-in item: %+v", item)
	}
	if item := findItem(d.Plans[1].Items, "Check-in"); item == nil || !item.Included {
		t.Fatalf("premium check-in item: %+v", item)
	}

	// Fakta & copy yang bergantung paket.
	if d.Facts.MinPrice != 100_000 || d.Facts.MaxGuests != 500 || d.Facts.GraceDays != 7 || !d.Facts.AllPlansDomain || d.Facts.AllPlansCheckin {
		t.Fatalf("facts: %+v", d.Facts)
	}
	checkin := findFeature(d.Features, "qr")
	if checkin == nil || checkin.Note != "Tersedia di Paket Premium" {
		t.Fatalf("check-in feature: %+v", checkin)
	}
	if !hasFAQ(d.FAQ, "check-in") || !hasFAQ(d.FAQ, "dibantu") {
		t.Fatalf("faq missing check-in/bantuan: %+v", d.FAQ)
	}
	if len(d.Steps) != 4 {
		t.Fatalf("steps: %d", len(d.Steps))
	}
}

func TestLandingCachesForTTL(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	th, pl := &fakeThemes{items: sampleThemes()}, &fakePlans{items: samplePlans()}
	s := newTestService(th, pl, nil, &now)
	ctx := context.Background()

	for range 3 {
		if _, err := s.Landing(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if th.calls != 1 || pl.calls != 1 {
		t.Fatalf("expected cached: themes=%d plans=%d", th.calls, pl.calls)
	}
	now = now.Add(CacheTTL + time.Second)
	if _, err := s.Landing(ctx); err != nil {
		t.Fatal(err)
	}
	if th.calls != 2 || pl.calls != 2 {
		t.Fatalf("expected refresh after TTL: themes=%d plans=%d", th.calls, pl.calls)
	}
}

func TestLandingWithoutWhatsApp(t *testing.T) {
	now := time.Now()
	for name, ct := range map[string]domain.ContactProvider{
		"nil provider": nil,
		"empty number": &fakeContact{},
		"invalid":      &fakeContact{c: domain.Contact{AdminWhatsApp: "12"}},
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestService(&fakeThemes{}, &fakePlans{items: samplePlans()}, ct, &now)
			d, err := s.Landing(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if d.WhatsAppURL != "" || d.WhatsAppDisplay != "" || hasFAQ(d.FAQ, "dibantu") {
				t.Fatalf("expected no WA: %q %q", d.WhatsAppURL, d.WhatsAppDisplay)
			}
		})
	}
}

func TestLandingDegradedKeepsLastGoodData(t *testing.T) {
	now := time.Now()
	th, pl := &fakeThemes{items: sampleThemes()}, &fakePlans{items: samplePlans()}
	s := newTestService(th, pl, nil, &now)
	ctx := context.Background()

	good, _ := s.Landing(ctx)
	now = now.Add(CacheTTL + time.Second)
	pl.err, pl.items = errors.New("db down"), nil
	d, err := s.Landing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d != good || len(d.Plans) != 2 {
		t.Fatalf("expected stale good data, got degraded=%v plans=%d", d.Degraded, len(d.Plans))
	}

	// Tanpa data lama: halaman tetap dibangun, paket kosong, Degraded=true.
	s2 := newTestService(&fakeThemes{items: sampleThemes()}, &fakePlans{err: errors.New("db down")}, &fakeContact{err: errors.New("x")}, &now)
	d2, err := s2.Landing(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !d2.Degraded || len(d2.Plans) != 0 || len(d2.Themes) != 4 || findFeature(d2.Features, "qr") != nil {
		t.Fatalf("degraded: %+v", d2)
	}
}

func TestSiteURL(t *testing.T) {
	cases := map[string]string{"undangin.id": "https://undangin.id", "localhost": "http://localhost", "": "http://localhost", "app.localhost": "http://app.localhost"}
	for in, want := range cases {
		if got := SiteURL(in); got != want {
			t.Errorf("SiteURL(%q)=%q want %q", in, got, want)
		}
	}
}

func findItem(items []domain.PlanItem, prefix string) *domain.PlanItem {
	for i := range items {
		if strings.HasPrefix(items[i].Text, prefix) {
			return &items[i]
		}
	}
	return nil
}

func findFeature(fs []domain.Feature, icon string) *domain.Feature {
	for i := range fs {
		if fs[i].Icon == icon {
			return &fs[i]
		}
	}
	return nil
}

func hasFAQ(items []domain.FAQItem, sub string) bool {
	for _, f := range items {
		if strings.Contains(strings.ToLower(f.Q), strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
