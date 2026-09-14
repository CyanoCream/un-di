// landingdev = server pengembangan landing page tanpa database (data contoh).
//
//	go run ./cmd/landingdev                          → http://localhost:8095/
//	WA= go run ./cmd/landingdev                      → tanpa nomor WhatsApp admin (section bantuan tersembunyi)
//	PLANS=1 go run ./cmd/landingdev                  → hanya satu paket (tanpa check-in)
//	THUMBS_DIR=/path go run ./cmd/landingdev         → thumbnail dari <dir>/<slug>/thumb.webp (selain themes/<slug>/assets)
//	THEMEDEV_URL=http://localhost:8090 go run ...    → /_preview/<slug> diproksikan ke cmd/themedev
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	landingctl "undangan/services/landing/controller"
	"undangan/services/landing/domain"
	landingsvc "undangan/services/landing/service"
)

func main() {
	addr := env("ADDR", ":8095")
	themesDir := env("THEMES_DIR", "themes")
	thumbsDir := os.Getenv("THUMBS_DIR")
	webDir := env("LANDING_DIR", filepath.Join("web", "landing"))

	wa := "6281234567890"
	if v, ok := os.LookupEnv("WA"); ok {
		wa = v
	}
	plans := demoPlans()
	if os.Getenv("PLANS") == "1" {
		plans = plans[:1]
	}

	thumb := func(slug string) (string, bool) {
		if thumbsDir != "" {
			if p := filepath.Join(thumbsDir, slug, "thumb.webp"); exists(p) {
				return p, true
			}
		}
		if p := filepath.Join(themesDir, slug, "assets", "thumb.webp"); exists(p) {
			return p, true
		}
		return "", false
	}

	themes := demoThemes()
	for i := range themes {
		if _, ok := thumb(themes[i].Slug); ok {
			themes[i].Thumbnail = "/_theme/" + themes[i].Slug + "/thumb.webp"
		}
	}

	svc := landingsvc.New(
		fakeThemes(themes), fakePlans(plans), fakeContact{wa: wa, merchant: "Undangin Digital"},
		landingsvc.Config{AppName: env("APP_NAME", "Undangin"), PortalURL: env("PORTAL_URL", "http://localhost:5173"), BaseDomain: env("BASE_DOMAIN", "undangin.id")},
	)
	ctl := landingctl.New(svc, webDir, true)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_theme/{theme}/{path...}", func(w http.ResponseWriter, r *http.Request) {
		slug, p := r.PathValue("theme"), r.PathValue("path")
		if p == "thumb.webp" {
			if file, ok := thumb(slug); ok {
				http.ServeFile(w, r, file)
				return
			}
		}
		http.ServeFile(w, r, filepath.Join(themesDir, filepath.Base(slug), "assets", filepath.Clean("/"+p)))
	})
	mux.HandleFunc("GET /_shared/{path...}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(themesDir, "_shared", filepath.Clean("/"+r.PathValue("path"))))
	})
	if base := os.Getenv("THEMEDEV_URL"); base != "" {
		target, err := url.Parse(base)
		if err != nil {
			log.Fatal(err)
		}
		proxy := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.URL.Path = "/" + strings.TrimPrefix(pr.In.URL.Path, "/_preview/")
			pr.Out.URL.RawQuery = "preview=1"
		}}
		mux.Handle("GET /_preview/{theme}", proxy)
	} else {
		mux.HandleFunc("GET /_preview/{theme}", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<!doctype html><meta charset=utf-8><meta name=viewport content="width=device-width"><body style="margin:0;display:grid;place-items:center;height:100vh;font-family:system-ui;background:#fbf7f2;color:#3a2a35;text-align:center;padding:1rem">
<div><p style="font-size:.8rem;letter-spacing:.2em;text-transform:uppercase">Pratinjau</p><h1>%s</h1><p>Set THEMEDEV_URL untuk melihat tema asli.</p></div>`, r.PathValue("theme"))
		})
	}
	mux.Handle("/", ctl)

	fmt.Printf("landingdev: http://localhost%s/\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

type fakeThemes []domain.Theme

func (f fakeThemes) ListThemes(context.Context) ([]domain.Theme, error) { return f, nil }

type fakePlans []domain.Plan

func (f fakePlans) ListPlans(context.Context) ([]domain.Plan, error) { return f, nil }

type fakeContact struct{ wa, merchant string }

func (f fakeContact) Contact(context.Context) (domain.Contact, error) {
	return domain.Contact{AdminWhatsApp: f.wa, MerchantName: f.merchant}, nil
}

func demoPlans() []domain.Plan {
	return []domain.Plan{
		{ID: "basic", Name: "Paket Basic", Description: "1 undangan digital, hingga 100 tamu, bisa pakai domain sendiri, aktif 30 hari",
			Price: 100_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 100, AllowCustomDomain: true},
		{ID: "premium", Name: "Paket Premium", Description: "1 undangan digital, hingga 500 tamu, check-in QR di venue, custom domain, aktif 30 hari",
			Price: 175_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 500, AllowCustomDomain: true, AllowCheckin: true},
	}
}

// demoThemes mengikuti katalog docs/THEMES.md (warna perkiraan).
func demoThemes() []domain.Theme {
	t := func(slug, name, cat, desc string, colors ...string) domain.Theme {
		return domain.Theme{Slug: slug, Name: name, Category: cat, Description: desc, Colors: colors}
	}
	return []domain.Theme{
		t("jawa-sogan", "Jawa Sogan", "adat", "Klasik Jawa bernuansa coklat sogan dan emas dengan motif kawung dan parang.", "#5a3825", "#c9a15b", "#f5ecdc", "#24150c"),
		t("islami-zamrud", "Islami Zamrud", "islami", "Hijau zamrud dan emas dengan pola bintang geometris, bingkai mihrab, dan lentera.", "#0f4d3a", "#c8a24a", "#faf6ec", "#0a3328"),
		t("floral-sage", "Floral Sage", "floral", "Hijau sage dan merah muda pudar di atas kertas krem, dihiasi bunga line-art.", "#8a9a7b", "#d9a7a0", "#f6f0e6", "#5d6e52"),
		t("minimalis-monokrom", "Minimalis Monokrom", "modern", "Editorial off-white dan hitam dengan tipografi serif besar dan garis tipis.", "#f4f2ee", "#151515", "#8a8783", "#d8d4cd"),
		t("royal-navy", "Royal Navy", "elegan", "Navy pekat dengan emas sampanye, garis art-deco, dan kipas sunburst.", "#0d1830", "#d6bd8f", "#1d3363", "#f4ecdc"),
		t("bali-kamboja", "Bali Kamboja", "adat", "Terakota dan gading dengan bunga kamboja; pembuka gapura candi bentar.", "#b5573a", "#f3ead8", "#5f6b3a"),
		t("minang-gadang", "Minang Gadang", "adat", "Marun dan emas songket berlapis dengan lengkung atap gonjong.", "#6d1a24", "#d4a646", "#1b1113"),
		t("batak-gorga", "Batak Gorga", "adat", "Poster tipografi tebal merah, hitam, dan putih tulang bermotif gorga.", "#9e1b1b", "#f1e9dc", "#141010"),
		t("sunda-siger", "Sunda Siger", "adat", "Mint dan kuning gading bergaya gulungan naskah, mahkota siger, dan bambu.", "#9fd3c0", "#f2e3b3", "#2f5d50"),
		t("betawi-gigi-balang", "Betawi Gigi Balang", "adat", "Grid kartu ceria merah cabai dan hijau pucuk dengan lis gigi balang.", "#d6362f", "#7dbb3f", "#f6c945"),
		t("midnight-burgundy", "Midnight Burgundy", "elegan", "Sinematik layar penuh: arang, burgundy, dan emas tua.", "#1c1a1b", "#6b1830", "#b08a4a"),
		t("champagne-marble", "Champagne Marble", "elegan", "Kartu akrilik bertumpuk di atas marmer putih dan rose gold.", "#f4f1ec", "#d9c3a5", "#b77f73"),
		t("terracotta-boho", "Terracotta Boho", "rustic", "Kolase asimetris terakota dan pasir dengan matahari bulat dan pampas.", "#c0633f", "#e8d5bb", "#6b4a35"),
		t("groovy-70an", "Groovy 70-an", "retro", "Bergelombang mustard dan oranye bakar dengan piringan hitam berputar.", "#e0a526", "#d4572a", "#5a3a22"),
		t("majalah-cinta", "Majalah Cinta", "modern", "Sampul dan spread majalah: The Wedding Issue.", "#d7263d", "#111111", "#ffffff"),
		t("lavender-awan", "Lavender Awan", "pastel", "Kartu melayang di langit lilac dan biru bedak.", "#c7b8e6", "#b9d3ee", "#fbf6ee"),
		t("tropis-monstera", "Tropis Monstera", "floral", "Section diagonal hijau tropis dan koral dengan daun monstera.", "#1f5e45", "#f08a6b", "#f7efe2"),
		t("scrapbook-polaroid", "Scrapbook Polaroid", "rustic", "Kolase polaroid miring di atas kertas kraft dengan selotip.", "#c8a882", "#3d5a80", "#f2d15e"),
		t("surat-lilin", "Surat Segel Lilin", "elegan", "Lembar surat bertumpuk di atas vellum dengan segel lilin merah.", "#eef0ec", "#7d93a8", "#a4282f"),
		t("nur-pastel", "Nur Pastel", "islami", "Kolom sempit blush dan krem dengan medali roset dan bulan sabit.", "#f3d3d3", "#f8efe4", "#c9a86a"),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func exists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
