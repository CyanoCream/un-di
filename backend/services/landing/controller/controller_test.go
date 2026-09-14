package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"undangan/services/landing/domain"
	landingsvc "undangan/services/landing/service"
)

var templatesDir = filepath.Join("..", "..", "..", "web", "landing")

type themes []domain.Theme

func (f themes) ListThemes(context.Context) ([]domain.Theme, error) { return f, nil }

type plans []domain.Plan

func (f plans) ListPlans(context.Context) ([]domain.Plan, error) { return f, nil }

type contact string

func (c contact) Contact(context.Context) (domain.Contact, error) {
	return domain.Contact{AdminWhatsApp: string(c), MerchantName: "Undangin"}, nil
}

func newController(t *testing.T, wa string, dev bool) *Controller {
	t.Helper()
	ths := themes{
		{Slug: "jawa-sogan", Name: "Jawa Sogan", Category: "adat", Description: "Klasik <Jawa>", Colors: []string{"#5a3825", "#c9a15b"}, Thumbnail: "/_theme/jawa-sogan/thumb.webp"},
		{Slug: "islami-zamrud", Name: "Islami Zamrud", Category: "islami", Colors: []string{"#0f4d3a"}},
		{Slug: "royal-navy", Name: "Royal Navy", Category: "elegan"},
	}
	for i := range 9 { // total > 8 untuk menguji data-extra di halaman utama
		ths = append(ths, domain.Theme{Slug: "tema-" + string(rune('a'+i)), Name: "Tema " + string(rune('A'+i)), Category: "modern"})
	}
	pls := plans{
		{ID: "basic", Name: "Paket Basic", Price: 100_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 100, AllowCustomDomain: true},
		{ID: "premium", Name: "Paket Premium", Price: 175_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 500, AllowCustomDomain: true, AllowCheckin: true},
	}
	svc := landingsvc.New(ths, pls, contact(wa), landingsvc.Config{AppName: "Undangin", PortalURL: "https://app.undangin.id", BaseDomain: "undangin.id"})
	return New(svc, templatesDir, dev)
}

func do(t *testing.T, h http.Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.Host = "undangin.id"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestLandingPage(t *testing.T) {
	c := newController(t, "6281234567890", false)
	rec := do(t, c, http.MethodGet, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		"<title>Undangin", `rel="canonical" href="https://undangin.id/"`, `property="og:title"`,
		"Rp100.000", "Rp175.000", "Paling hemat per tamu",
		"Jawa Sogan", "Islami Zamrud", "Royal Navy", "Klasik &lt;Jawa&gt;",
		`href="https://app.undangin.id/daftar"`, `href="https://app.undangin.id/masuk"`,
		`href="https://app.undangin.id/daftar?tema=jawa-sogan"`, `href="/_preview/jawa-sogan"`,
		"https://wa.me/6281234567890?text=", "Butuh dibantu?", `id="tema"`, `id="harga"`, `id="faq"`,
		`src="/_theme/jawa-sogan/thumb.webp"`, "Tersedia di Paket Premium", "data-extra",
		`/_landing/landing.css?v=`, "application/ld+json",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	if strings.Contains(body, "ZgotmplZ") {
		t.Error("template produced ZgotmplZ (unsafe value filtered)")
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "max-age=60") {
		t.Errorf("cache-control %q", got)
	}
	if rec.Header().Get("Content-Security-Policy") == "" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing security headers")
	}
	checkJSONLD(t, body)
}

func TestLandingWithoutWhatsApp(t *testing.T) {
	c := newController(t, "", false)
	body := do(t, c, http.MethodGet, "/").Body.String()
	if strings.Contains(body, "wa.me") || strings.Contains(body, "Butuh dibantu?") {
		t.Fatal("WhatsApp CTA must be hidden when admin number is empty")
	}
	if !strings.Contains(body, "Rp175.000") {
		t.Fatal("pricing should still render")
	}
}

func TestThemeCatalogPage(t *testing.T) {
	c := newController(t, "6281234567890", false)
	rec := do(t, c, http.MethodGet, "/tema?kategori=adat")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Katalog tema", `rel="canonical" href="https://undangin.id/tema"`, "Jawa Sogan", "Royal Navy",
		"Menampilkan 1 tema kategori Adat Nusantara", `href="/tema?kategori=islami"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	// Royal Navy (elegan) tersembunyi saat filter adat; Jawa Sogan tidak.
	if !regexp.MustCompile(`data-cat="elegan" hidden`).MatchString(body) || strings.Contains(body, `data-cat="adat" hidden`) {
		t.Error("category filter not applied server-side")
	}
	if strings.Contains(body, "data-extra") {
		t.Error("catalog page must show all themes")
	}
	if do(t, c, http.MethodGet, "/tema/").Code != http.StatusMovedPermanently {
		t.Error("/tema/ should redirect")
	}
}

func TestRobotsSitemapAssetsAnd404(t *testing.T) {
	c := newController(t, "6281234567890", false)

	rec := do(t, c, http.MethodGet, "/robots.txt")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Sitemap: https://undangin.id/sitemap.xml") || !strings.Contains(rec.Body.String(), "Disallow: /api/") {
		t.Fatalf("robots: %d %s", rec.Code, rec.Body.String())
	}

	rec = do(t, c, http.MethodGet, "/sitemap.xml")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "<loc>https://undangin.id/tema</loc>") || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/xml") {
		t.Fatalf("sitemap: %d %s", rec.Code, rec.Body.String())
	}

	rec = do(t, c, http.MethodGet, "/_landing/landing.css?v=abc")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") || !strings.Contains(rec.Header().Get("Content-Type"), "text/css") {
		t.Fatalf("css asset: %d %v", rec.Code, rec.Header())
	}
	rec = do(t, c, http.MethodGet, "/_landing/favicon.svg")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/svg+xml" {
		t.Fatalf("svg asset: %d %v", rec.Code, rec.Header())
	}
	for _, p := range []string{"/_landing/nope.css", "/_landing/../layout.html", "/_landing/", "/tidak-ada", "/wp-admin"} {
		rec = do(t, c, http.MethodGet, p)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", p, rec.Code)
		}
		if p == "/tidak-ada" && (!strings.Contains(rec.Body.String(), "salah alamat") || !strings.Contains(rec.Body.String(), `name="robots" content="noindex"`)) {
			t.Error("404 page should be the styled landing 404 with noindex")
		}
	}

	if rec = do(t, c, http.MethodPost, "/"); rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST /: %d", rec.Code)
	}
	if rec = do(t, c, http.MethodHead, "/"); rec.Code != 200 || rec.Body.Len() != 0 {
		t.Errorf("HEAD /: %d len=%d", rec.Code, rec.Body.Len())
	}
}

func TestDevModeOriginAndNoCache(t *testing.T) {
	svc := landingsvc.New(themes{}, plans{}, nil, landingsvc.Config{AppName: "Undangin", PortalURL: "http://localhost:5173", BaseDomain: "localhost"})
	c := New(svc, templatesDir, true)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	c.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="http://localhost:8080/"`) || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("dev origin/cache: %s", rec.Header().Get("Cache-Control"))
	}
	// Tanpa tema & paket: section disembunyikan, halaman tetap utuh.
	if strings.Contains(body, `id="harga"`) || strings.Contains(body, `id="tema"`) || !strings.Contains(body, `id="faq"`) {
		t.Error("empty sections should be hidden")
	}
}

func checkJSONLD(t *testing.T, body string) {
	t.Helper()
	m := regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`).FindStringSubmatch(body)
	if m == nil {
		t.Fatal("no JSON-LD")
	}
	var doc struct {
		Graph []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(m[1]), &doc); err != nil {
		t.Fatalf("JSON-LD invalid: %v\n%s", err, m[1])
	}
	types := map[string]map[string]any{}
	for _, g := range doc.Graph {
		types[g["@type"].(string)] = g
	}
	for _, want := range []string{"Organization", "WebSite", "Product", "FAQPage"} {
		if types[want] == nil {
			t.Errorf("JSON-LD missing %s", want)
		}
	}
	if offers, _ := types["Product"]["offers"].([]any); len(offers) != 2 {
		t.Errorf("Product offers: %v", types["Product"]["offers"])
	}
	if strings.Contains(m[1], "aggregateRating") || strings.Contains(m[1], "review") {
		t.Error("JSON-LD must not contain ratings/reviews")
	}
}
