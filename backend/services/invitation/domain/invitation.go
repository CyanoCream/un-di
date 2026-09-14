// Package domain (invitation) = aggregate Undangan beserta Tamu & Ucapan, aturan bisnis, dan kontrak repository.
package domain

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"undangan/kernel/apperror"
)

type Status string

const (
	StatusDraft     Status = "draft"
	StatusPublished Status = "published"
	StatusSuspended Status = "suspended"
)

// Mode akses halaman undangan (docs/SPEC.md §8).
const (
	AccessPublic    = "public"
	AccessGuestOnly = "guest_only"
)

type Invitation struct {
	ID            string
	UserID        string
	OwnerName     string
	OwnerEmail    string
	Theme         *string
	ThemeLockedAt *time.Time
	Status        Status
	Content       Content
	Subdomain     *string
	// Pengaturan akses & check-in
	AccessMode     string
	CheckinEnabled bool
	CheckinPinHash string
	CustomDomain   *CustomDomain // nil bila belum ditambahkan
	GuestCount     int
	PublishedAt    *time.Time
	SuspendedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (i *Invitation) ThemeLocked() bool { return i.ThemeLockedAt != nil }

// Title: "Raka & Nadia" sesuai urutan mempelai.
func (i *Invitation) Title() string {
	c := i.Content
	name := func(p Person) string {
		if p.Nickname != "" {
			return p.Nickname
		}
		return p.FullName
	}
	first, second := name(c.Groom), name(c.Bride)
	if c.CoupleOrder == "bride_first" {
		first, second = second, first
	}
	if first == "" && second == "" {
		return "(belum diisi)"
	}
	return strings.TrimSpace(strings.Trim(first+" & "+second, " &"))
}

// EventDate = tanggal acara pertama yang terisi.
func (i *Invitation) EventDate() *string {
	for _, e := range i.Content.Events {
		if e.Date != "" {
			d := e.Date
			return &d
		}
	}
	return nil
}

// NewContent = konten awal saat undangan dibuat.
func NewContent(eventType string) Content {
	c := Content{
		EventType:   "pernikahan",
		Religion:    "islam",
		CoupleOrder: "groom_first",
		RSVP:        RSVP{Enabled: true, MaxPax: 2, ShowWishes: true},
		Gift:        Gift{Accounts: []GiftAccount{}},
	}
	if eventType == "ngunduh_mantu" {
		c.EventType = eventType
		c.Events = []Event{{ID: NewID(), Name: "Ngunduh Mantu", Timezone: "WIB"}}
	} else {
		c.Events = []Event{
			{ID: NewID(), Name: "Akad Nikah", Timezone: "WIB"},
			{ID: NewID(), Name: "Resepsi", Timezone: "WIB"},
		}
	}
	return NormalizeContent(c)
}

// NormalizeContent memastikan slice tidak nil (JSON [] bukan null).
func NormalizeContent(c Content) Content {
	if c.Events == nil {
		c.Events = []Event{}
	}
	if c.LoveStory == nil {
		c.LoveStory = []Story{}
	}
	if c.Gallery.Photos == nil {
		c.Gallery.Photos = []string{}
	}
	if c.Gift.Accounts == nil {
		c.Gift.Accounts = []GiftAccount{}
	}
	if c.Closing.Family == nil {
		c.Closing.Family = []string{}
	}
	return c
}

var (
	dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)
)

func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

// validURL: kosong, http(s)://, atau path upload lokal.
func validURL(s string) bool {
	if s == "" || strings.HasPrefix(s, "/uploads/") {
		return !strings.Contains(s, "..")
	}
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// ValidateContent memeriksa batas & format; string dirapikan (trim + panjang maksimum).
func ValidateContent(c Content) (Content, error) {
	f := apperror.Fields{}
	t := func(s *string, max int) {
		*s = strings.TrimSpace(*s)
		if r := []rune(*s); len(r) > max {
			*s = string(r[:max])
		}
	}
	checkURL := func(field string, s *string) {
		t(s, 2048)
		if !validURL(*s) {
			f.Add(field, "URL tidak valid (harus diawali https://)")
		}
	}

	if c.EventType == "" {
		c.EventType = "pernikahan"
	}
	if !oneOf(c.EventType, "pernikahan", "ngunduh_mantu") {
		f.Add("event_type", "Jenis acara tidak valid")
	}
	if c.Religion != "" && !oneOf(c.Religion, "islam", "kristen", "katolik", "hindu", "buddha", "konghucu", "umum") {
		f.Add("religion", "Agama tidak valid")
	}
	if c.CoupleOrder != "" && !oneOf(c.CoupleOrder, "groom_first", "bride_first") {
		f.Add("couple_order", "Urutan mempelai tidak valid")
	}

	t(&c.Cover.Title, 80)
	checkURL("cover.photo", &c.Cover.Photo)
	checkURL("cover.background", &c.Cover.Background)
	t(&c.Opening.Greeting, 200)
	t(&c.Opening.Text, 2000)
	t(&c.Quote.Text, 1000)
	t(&c.Quote.Source, 200)

	for key, p := range map[string]*Person{"groom": &c.Groom, "bride": &c.Bride} {
		t(&p.FullName, 150)
		t(&p.Nickname, 50)
		t(&p.ChildOrder, 50)
		t(&p.Father, 150)
		t(&p.Mother, 150)
		p.Instagram = strings.TrimPrefix(strings.TrimSpace(p.Instagram), "@")
		t(&p.Instagram, 60)
		checkURL(key+".photo", &p.Photo)
	}

	if len(c.Events) > 10 {
		f.Add("events", "Maksimal 10 acara")
	}
	for i := range c.Events {
		e := &c.Events[i]
		k := fmt.Sprintf("events.%d.", i)
		if e.ID == "" {
			e.ID = NewID()
		}
		t(&e.Name, 80)
		t(&e.Venue, 200)
		t(&e.Address, 500)
		t(&e.Note, 500)
		if e.Date != "" {
			if _, err := time.Parse("2006-01-02", e.Date); err != nil || !dateRe.MatchString(e.Date) {
				f.Add(k+"date", "Tanggal tidak valid")
			}
		}
		if e.StartTime != "" && !timeRe.MatchString(e.StartTime) {
			f.Add(k+"start_time", "Jam mulai tidak valid (HH:MM)")
		}
		if e.EndTime != "" && !timeRe.MatchString(e.EndTime) {
			f.Add(k+"end_time", "Jam selesai tidak valid (HH:MM)")
		}
		if e.Timezone == "" {
			e.Timezone = "WIB"
		}
		if !oneOf(e.Timezone, "WIB", "WITA", "WIT") {
			f.Add(k+"timezone", "Zona waktu harus WIB, WITA, atau WIT")
		}
		checkURL(k+"maps_url", &e.MapsURL)
	}

	if len(c.LoveStory) > 20 {
		f.Add("love_story", "Maksimal 20 cerita")
	}
	for i := range c.LoveStory {
		s := &c.LoveStory[i]
		t(&s.Date, 50)
		t(&s.Title, 100)
		t(&s.Text, 2000)
		checkURL(fmt.Sprintf("love_story.%d.photo", i), &s.Photo)
	}

	if len(c.Gallery.Photos) > 20 {
		f.Add("gallery.photos", "Maksimal 20 foto galeri")
	}
	for i := range c.Gallery.Photos {
		checkURL(fmt.Sprintf("gallery.photos.%d", i), &c.Gallery.Photos[i])
	}
	checkURL("gallery.video_url", &c.Gallery.VideoURL)
	checkURL("live_stream.url", &c.LiveStream.URL)
	t(&c.LiveStream.Platform, 50)
	t(&c.LiveStream.Note, 300)

	t(&c.Gift.Text, 1000)
	if len(c.Gift.Accounts) > 10 {
		f.Add("gift.accounts", "Maksimal 10 rekening")
	}
	for i := range c.Gift.Accounts {
		a := &c.Gift.Accounts[i]
		if a.Type != "ewallet" {
			a.Type = "bank"
		}
		t(&a.Provider, 50)
		t(&a.Number, 50)
		t(&a.Holder, 100)
	}
	t(&c.Gift.Address.Recipient, 100)
	t(&c.Gift.Address.Phone, 30)
	t(&c.Gift.Address.Address, 500)

	if c.RSVP.MaxPax < 1 || c.RSVP.MaxPax > 10 {
		c.RSVP.MaxPax = 2
	}

	t(&c.Closing.Text, 2000)
	t(&c.Closing.SignOff, 200)
	t(&c.Closing.From, 300)
	if len(c.Closing.Family) > 20 {
		f.Add("closing.family", "Maksimal 20 baris turut mengundang")
	}
	for i := range c.Closing.Family {
		t(&c.Closing.Family[i], 200)
	}

	checkURL("music.url", &c.Music.URL)
	t(&c.Music.Title, 100)
	if c.Music.StartAt < 0 || c.Music.StartAt > 3600 {
		c.Music.StartAt = 0
	}
	t(&c.Share.WhatsAppTemplate, 2000)

	if err := f.Err(); err != nil {
		return c, err
	}
	return NormalizeContent(c), nil
}

// PublishChecklist mengembalikan syarat yang belum terpenuhi (kosong = boleh publish).
func PublishChecklist(inv *Invitation, hasSubscription bool) apperror.Fields {
	f := apperror.Fields{}
	if inv.Theme == nil {
		f.Add("theme", "Pilih tema terlebih dahulu")
	}
	if inv.Subdomain == nil {
		f.Add("subdomain", "Atur alamat (subdomain) undangan")
	}
	c := inv.Content
	if c.Groom.FullName == "" && c.Groom.Nickname == "" {
		f.Add("groom", "Isi nama mempelai pria")
	}
	if c.Bride.FullName == "" && c.Bride.Nickname == "" {
		f.Add("bride", "Isi nama mempelai wanita")
	}
	if inv.EventDate() == nil {
		f.Add("events", "Isi minimal satu acara beserta tanggalnya")
	}
	if !hasSubscription {
		f.Add("subscription", "Paket langganan tidak aktif")
	}
	return f
}

// NewID = UUID v4.
func NewID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
