// Package controller (landing) menyajikan halaman pemasaran di host platform (apex/www):
//
//	GET /                 landing
//	GET /tema             katalog tema lengkap (?kategori=adat)
//	GET /robots.txt       robots
//	GET /sitemap.xml      sitemap
//	GET /_landing/{path}  aset statis dari <templatesDir>/assets
//	lainnya               404 bergaya landing
package controller

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"undangan/services/landing/domain"
	landingsvc "undangan/services/landing/service"
	"undangan/kernel/storage"
)

const assetPrefix = "/_landing/"

type Controller struct {
	svc    landingsvc.Service
	dir    string
	dev    bool
	assets fs.FS

	mu       sync.Mutex // menjaga pages
	pages    map[string]*template.Template
	vmu      sync.Mutex // menjaga versions
	versions map[string]string
}

// New: templatesDir berisi layout.html, index.html, tema.html, 404.html, dan assets/.
// dev=true → template & hash aset dibaca ulang setiap request, tanpa cache browser.
func New(svc landingsvc.Service, templatesDir string, dev bool) *Controller {
	return &Controller{
		svc: svc, dir: templatesDir, dev: dev,
		assets:   storage.NoDirFS{FS: os.DirFS(filepath.Join(templatesDir, "assets"))},
		pages:    map[string]*template.Template{},
		versions: map[string]string{},
	}
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}
	p := r.URL.Path
	switch {
	case p == "/" || p == "/index.html":
		c.renderPage(w, r, "index", http.StatusOK)
	case p == "/tema":
		c.renderPage(w, r, "tema", http.StatusOK)
	case p == "/tema/":
		http.Redirect(w, r, "/tema", http.StatusMovedPermanently)
	case p == "/robots.txt":
		c.robots(w, r)
	case p == "/sitemap.xml":
		c.sitemap(w, r)
	case p == "/favicon.ico":
		http.Redirect(w, r, assetPrefix+"favicon.svg", http.StatusMovedPermanently)
	case strings.HasPrefix(p, assetPrefix):
		c.asset(w, r, strings.TrimPrefix(p, assetPrefix))
	default:
		c.renderPage(w, r, "404", http.StatusNotFound)
	}
}

// ---- halaman ----

type pageData struct {
	*domain.LandingData
	Page        string
	Title       string
	Description string
	Origin      string
	Canonical   string
	OGImage     string
	JSONLD      template.JS
	Dev         bool

	ThemeLimit     int    // kartu tema di halaman utama sebelum "Lihat semua"
	ActiveCategory string // /tema?kategori=
	ActiveLabel    string
	Visible        int
}

func (c *Controller) renderPage(w http.ResponseWriter, r *http.Request, name string, status int) {
	tpl, err := c.template(name)
	if err != nil {
		slog.Error("landing: template", "page", name, "err", err)
		http.Error(w, "Halaman sedang tidak tersedia", http.StatusInternalServerError)
		return
	}
	data, err := c.svc.Landing(r.Context())
	if err != nil || data == nil {
		slog.Error("landing: data", "err", err)
		http.Error(w, "Halaman sedang tidak tersedia", http.StatusServiceUnavailable)
		return
	}
	pd := c.buildPage(r, name, data)

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "layout", pd); err != nil {
		slog.Error("landing: render", "page", name, "err", err)
		http.Error(w, "Halaman sedang tidak tersedia", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	securityHeaders(h)
	switch {
	case c.dev:
		h.Set("Cache-Control", "no-store")
	case status == http.StatusOK:
		h.Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	default:
		h.Set("Cache-Control", "no-cache")
	}
	h.Set("Content-Length", fmt.Sprint(buf.Len()))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(buf.Bytes())
	}
}

func (c *Controller) buildPage(r *http.Request, name string, d *domain.LandingData) pageData {
	origin := c.origin(r, d)
	pd := pageData{LandingData: d, Page: name, Origin: origin, Dev: c.dev, ThemeLimit: domain.LandingThemeLimit}
	switch name {
	case "index":
		pd.Canonical = origin + "/"
		pd.Title = d.Brand + " — Undangan Pernikahan Digital yang Elegan & Berguna di Hari-H"
		pd.Description = "Buat undangan pernikahan digital dengan tema original, link personal per tamu, RSVP, amplop digital, dan check-in QR di venue. Tanpa aplikasi, bayar via QRIS."
		if d.Facts.MinPrice > 0 {
			pd.Description = "Undangan pernikahan digital mulai " + domain.FormatRupiah(d.Facts.MinPrice) + ": tema original, link personal per tamu, RSVP, amplop digital, dan check-in QR. Tanpa aplikasi, bayar via QRIS."
		}
		pd.JSONLD = jsonLD(d, origin, true)
	case "tema":
		pd.Canonical = origin + "/tema"
		pd.Title = "Katalog Tema Undangan Digital — " + d.Brand
		pd.Description = "Jelajahi tema undangan pernikahan digital " + d.Brand + ": adat Nusantara, Islami, floral, modern, elegan, dan lainnya. Coba demo langsung dari HP."
		if cat := r.URL.Query().Get("kategori"); domain.ValidCategory(cat) {
			for _, x := range d.Categories {
				if x.Key == cat {
					pd.ActiveCategory, pd.ActiveLabel = cat, x.Label
				}
			}
		}
		for _, t := range d.Themes {
			if pd.ActiveCategory == "" || t.Category == pd.ActiveCategory {
				pd.Visible++
			}
		}
		pd.JSONLD = jsonLD(d, origin, false)
	default:
		pd.Title = "Halaman tidak ditemukan — " + d.Brand
		pd.Description = "Halaman yang Anda cari tidak ada."
	}
	if d.HeroTheme != nil && d.HeroTheme.Thumbnail != "" && strings.HasPrefix(d.HeroTheme.Thumbnail, "/") {
		pd.OGImage = origin + d.HeroTheme.Thumbnail
	}
	return pd
}

// origin: saat dev/localhost memakai Host request (termasuk port) supaya tautan kanonik bisa diklik.
func (c *Controller) origin(r *http.Request, d *domain.LandingData) string {
	site := d.SiteURL
	if strings.HasPrefix(site, "http://") && r.Host != "" {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host == "localhost" || strings.HasSuffix(host, ".localhost") || net.ParseIP(host) != nil {
			return "http://" + r.Host
		}
	}
	return site
}

func (c *Controller) template(name string) (*template.Template, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if t, ok := c.pages[name]; ok && !c.dev {
		return t, nil
	}
	t, err := template.New("layout.html").Funcs(c.funcs()).ParseFiles(
		filepath.Join(c.dir, "layout.html"), filepath.Join(c.dir, name+".html"))
	if err != nil {
		return nil, err
	}
	c.pages[name] = t
	return t, nil
}

func (c *Controller) funcs() template.FuncMap {
	return template.FuncMap{
		"asset":    c.assetURL,
		"rupiah":   domain.FormatRupiah,
		"number":   domain.FormatNumber,
		"add":      func(a, b int) int { return a + b },
		"pad2":     func(n int) string { return fmt.Sprintf("%02d", n) },
		"first":    func(s []domain.ThemeView, n int) []domain.ThemeView { return s[:min(n, len(s))] },
		"cssColor": cssColor,
		// colorAt = warna ke-i atau fallback (palet tema bisa kurang dari 3 warna).
		"colorAt": func(colors []string, i int, fallback string) template.CSS {
			if i >= 0 && i < len(colors) {
				return cssColor(colors[i])
			}
			return cssColor(fallback)
		},
		"themeCard": func(t domain.ThemeView, registerURL string, extra bool, activeCategory string) themeCard {
			return themeCard{T: t, Extra: extra, Hidden: activeCategory != "" && t.Category != activeCategory,
				UseURL: registerURL + "?tema=" + url.QueryEscape(t.Slug)}
		},
	}
}

type themeCard struct {
	T      domain.ThemeView
	UseURL string
	Extra  bool // di luar LandingThemeLimit (disembunyikan saat filter "Semua" di halaman utama)
	Hidden bool // tidak cocok dengan ?kategori= di /tema
}

// cssColor hanya meloloskan hex (#rgb / #rrggbb); lainnya → transparent.
func cssColor(s string) template.CSS {
	if len(s) != 4 && len(s) != 7 || s[0] != '#' {
		return "transparent"
	}
	for _, r := range s[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return "transparent"
		}
	}
	return template.CSS(s)
}

// assetURL menambah ?v=<hash isi> supaya aset bisa di-cache lama.
func (c *Controller) assetURL(name string) string {
	u := assetPrefix + name
	if c.dev {
		return u
	}
	c.vmu.Lock()
	defer c.vmu.Unlock()
	if v, ok := c.versions[name]; ok {
		return u + "?v=" + v
	}
	b, err := fs.ReadFile(c.assets, name)
	if err != nil {
		return u
	}
	sum := sha256.Sum256(b)
	v := hex.EncodeToString(sum[:])[:10]
	c.versions[name] = v
	return u + "?v=" + v
}

// ---- aset, robots, sitemap ----

func (c *Controller) asset(w http.ResponseWriter, r *http.Request, name string) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if name == "" || strings.HasPrefix(path.Base(name), ".") {
		c.renderPage(w, r, "404", http.StatusNotFound)
		return
	}
	f, err := c.assets.Open(name)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			c.renderPage(w, r, "404", http.StatusNotFound)
			return
		}
		http.Error(w, "Gagal membaca aset", http.StatusInternalServerError)
		return
	}
	f.Close()
	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	switch {
	case c.dev:
		h.Set("Cache-Control", "no-cache")
	case r.URL.Query().Get("v") != "":
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	default:
		h.Set("Cache-Control", "public, max-age=86400")
	}
	if strings.HasSuffix(name, ".svg") {
		h.Set("Content-Type", "image/svg+xml")
	}
	http.ServeFileFS(w, r, c.assets, name)
}

func (c *Controller) robots(w http.ResponseWriter, r *http.Request) {
	origin := r.Host
	if d, err := c.svc.Landing(r.Context()); err == nil && d != nil {
		origin = c.origin(r, d)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", c.textCache())
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /api/\nDisallow: /_preview/\n\nSitemap: %s/sitemap.xml\n", origin)
}

func (c *Controller) sitemap(w http.ResponseWriter, r *http.Request) {
	d, err := c.svc.Landing(r.Context())
	if err != nil || d == nil {
		http.Error(w, "Sitemap sedang tidak tersedia", http.StatusServiceUnavailable)
		return
	}
	origin := c.origin(r, d)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range []struct{ loc, freq, prio string }{{"/", "weekly", "1.0"}, {"/tema", "weekly", "0.8"}} {
		fmt.Fprintf(&b, "  <url><loc>%s</loc><changefreq>%s</changefreq><priority>%s</priority></url>\n", xmlEscape(origin+u.loc), u.freq, u.prio)
	}
	b.WriteString("</urlset>\n")
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", c.textCache())
	_, _ = w.Write([]byte(b.String()))
}

func (c *Controller) textCache() string {
	if c.dev {
		return "no-cache"
	}
	return "public, max-age=3600"
}

func securityHeaders(h http.Header) {
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; "+
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; "+
		"script-src 'self'; frame-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
}

var xmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

func xmlEscape(s string) string { return xmlReplacer.Replace(s) }

// ---- JSON-LD ----

func jsonLD(d *domain.LandingData, origin string, withFAQ bool) template.JS {
	org := map[string]any{
		"@type": "Organization",
		"@id":   origin + "/#organization",
		"name":  d.Brand,
		"url":   origin + "/",
		"logo":  origin + assetPrefix + "logo.svg",
	}
	if d.WhatsAppURL != "" && d.WhatsAppDisplay != "" {
		org["contactPoint"] = map[string]any{
			"@type": "ContactPoint", "contactType": "customer support",
			"telephone": strings.NewReplacer(" ", "", "-", "").Replace(d.WhatsAppDisplay), "availableLanguage": "id",
		}
	}
	graph := []any{
		org,
		map[string]any{"@type": "WebSite", "@id": origin + "/#website", "url": origin + "/", "name": d.Brand, "inLanguage": "id-ID",
			"publisher": map[string]string{"@id": origin + "/#organization"}},
	}
	if len(d.Plans) > 0 {
		offers := make([]any, 0, len(d.Plans))
		for _, p := range d.Plans {
			offers = append(offers, map[string]any{
				"@type": "Offer", "name": p.Name, "price": fmt.Sprint(p.Price), "priceCurrency": "IDR",
				"availability": "https://schema.org/InStock", "url": d.RegisterURL,
			})
		}
		graph = append(graph, map[string]any{
			"@type": "Product", "name": "Undangan Pernikahan Digital " + d.Brand,
			"description": "Undangan pernikahan digital dengan tema original, link personal per tamu, RSVP, amplop digital, dan check-in QR.",
			"brand":       map[string]string{"@id": origin + "/#organization"},
			"offers":      offers,
		})
	}
	if withFAQ && len(d.FAQ) > 0 {
		items := make([]any, 0, len(d.FAQ))
		for _, f := range d.FAQ {
			items = append(items, map[string]any{"@type": "Question", "name": f.Q,
				"acceptedAnswer": map[string]string{"@type": "Answer", "text": f.A}})
		}
		graph = append(graph, map[string]any{"@type": "FAQPage", "mainEntity": items})
	}
	b, err := json.Marshal(map[string]any{"@context": "https://schema.org", "@graph": graph})
	if err != nil {
		return "{}"
	}
	return template.JS(b) // json.Marshal meng-escape < > & sehingga aman di <script>
}
