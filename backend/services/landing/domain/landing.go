// Package domain (landing) = view model halaman pemasaran publik (host platform) beserta
// port data lintas modul (katalog tema, paket, kontak admin). Murni: tanpa http/pgx.
package domain

import (
	"context"
	"time"
)

// ---- port lintas modul (diimplementasikan lewat adapter, lihat package landing/adapter) ----

// ThemeLister = katalog tema aktif, urut sesuai sort_order.
type ThemeLister interface {
	ListThemes(ctx context.Context) ([]Theme, error)
}

// PlanLister = paket aktif, urut sort_order, harga.
type PlanLister interface {
	ListPlans(ctx context.Context) ([]Plan, error)
}

// ContactProvider = kontak admin (dari pengaturan pembayaran).
type ContactProvider interface {
	Contact(ctx context.Context) (Contact, error)
}

// ---- data masukan ----

type Theme struct {
	Slug        string
	Name        string
	Description string
	Category    string // adat | islami | floral | modern | elegan | rustic | pastel | retro
	Thumbnail   string // kosong bila tema belum punya thumbnail
	PreviewURL  string // kosong → /_preview/<slug>
	Colors      []string
}

type Plan struct {
	ID                string
	Name              string
	Description       string
	Price             int64
	DurationDays      int
	GraceDays         int
	MaxInvitations    int
	MaxGuests         int
	AllowCustomDomain bool
	AllowCheckin      bool
}

type Contact struct {
	AdminWhatsApp string // 628xx; kosong = belum diatur
	MerchantName  string
}

// ---- view model ----

type LandingData struct {
	Brand       string
	SiteURL     string // origin kanonik, mis. https://undangin.id (tanpa garis miring akhir)
	BaseDomain  string
	PortalURL   string
	RegisterURL string
	LoginURL    string

	// WhatsAppURL kosong bila nomor admin belum diatur → section "Butuh dibantu?" disembunyikan.
	WhatsAppURL     string
	WhatsAppDisplay string // +62 812-3456-7890
	MerchantName    string

	Plans      []PlanView
	Themes     []ThemeView // semua tema aktif
	Categories []Category  // hanya kategori yang punya tema, urut kanonik
	HeroTheme  *ThemeView  // tema pertama yang punya thumbnail (nil → mockup CSS)

	Facts    Facts
	Features []Feature
	Steps    []Step
	FAQ      []FAQItem

	Year        int
	GeneratedAt time.Time
	// Degraded = sebagian data gagal dimuat (section terkait disembunyikan).
	Degraded bool
}

// Facts = angka nyata dari database yang dipakai copy (tanpa klaim karangan).
type Facts struct {
	ThemeCount        int
	PlanCount         int
	MinPrice          int64
	MaxGuests         int
	GraceDays         int      // masa tenggang terbesar di antara paket (default 7)
	CheckinPlans      []string // nama paket dengan check-in
	CustomDomainPlans []string // nama paket dengan domain sendiri
	AllPlansCheckin   bool
	AllPlansDomain    bool
}

type ThemeView struct {
	Slug          string
	Name          string
	Description   string
	Category      string
	CategoryLabel string
	Thumbnail     string
	PreviewURL    string
	Colors        []string
	Initials      string // untuk kartu tanpa thumbnail
}

type Category struct {
	Key   string
	Label string
	Count int
}

type PlanView struct {
	ID          string
	Name        string
	Description string
	Price       int64
	PriceLabel  string // "Rp175.000" / "Gratis"
	Duration    string // "30 hari"
	Highlight   bool
	Badge       string
	Items       []PlanItem
}

type PlanItem struct {
	Text     string
	Included bool
}

type Feature struct {
	Icon  string // id simbol SVG di layout (#i-<icon>)
	Title string
	Text  string
	Note  string // keterangan ketersediaan, mis. "Tersedia di Paket Premium"
	Wide  bool   // kartu unggulan (lebih lebar di desktop)
}

type Step struct {
	Title string
	Text  string
}

type FAQItem struct {
	Q string
	A string
}
