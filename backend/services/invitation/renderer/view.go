package renderer

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"undangan/services/invitation/domain"
)

// PageData = semua yang tersedia di template tema.
type PageData struct {
	C            domain.Content
	Couple       []PersonView // urut sesuai couple_order
	Events       []EventView
	FirstEvent   *EventView
	Guest        *GuestView // nil jika tidak pakai link personal
	GuestName    string     // dari guest atau ?to=
	InvitationID string     // untuk API ucapan publik
	Preview      bool       // true di iframe editor / katalog tema: form tidak mengirim data
	Theme        string     // /_theme/<slug>
	Shared       string     // /_shared
	URL          string     // canonical
	OGImage      string
	Title        string // "Budi & Ani"

	// Akses & fitur tambahan (lihat docs/SPEC.md §8–10)
	AccessMode  string        // public | guest_only
	Checkin     bool          // tiket check-in ditampilkan (butuh .Guest)
	GuestQR     template.HTML // SVG QR tiket (dibuat server; aman)
	GiftConfirm bool          // form konfirmasi hadiah aktif
}

type PersonView struct {
	domain.Person
	Role        string // groom | bride
	RoleLabel   string // "Mempelai Pria" | "Mempelai Wanita"
	FatherLabel string // "Bapak Suyadi (Alm.)"
	MotherLabel string
	ParentsLine string // "Bapak X & Ibu Y"
}

type EventView struct {
	domain.Event
	Start       time.Time
	DayName     string // Minggu
	Day         string // 12
	MonthName   string // Juli
	Year        string // 2026
	DateLong    string // Minggu, 12 Juli 2026
	TimeRange   string // 08.00 – Selesai WIB
	ISO         string // RFC3339 untuk countdown
	CalendarURL string // Google Calendar
	MapsLink    string // maps_url atau pencarian alamat
	IsPast      bool
}

type GuestView struct {
	Name string
	Pax  int
	Code string // passcode check-in
	Slug string
	Link string // link personal absolut/relatif
}

// WithAccess melengkapi PageData dengan mode akses & tiket check-in.
func (d PageData) WithAccess(accessMode string, checkinEnabled bool) PageData {
	d.AccessMode = accessMode
	if d.AccessMode == "" {
		d.AccessMode = "public"
	}
	if checkinEnabled && d.Guest != nil {
		d.Checkin = true
		link := d.Guest.Link
		if link == "" {
			link = d.Guest.Slug
		}
		d.GuestQR = QRSVG(link + "?c=" + d.Guest.Code)
	}
	return d
}

// GateData = halaman gerbang (mode khusus tamu) / nama tidak terdaftar.
type GateData struct {
	PageData
	Reason    string // guest_only | not_registered
	Attempted string // path yang dicoba (ditampilkan ter-escape)
	Error     string // mis. "Kode undangan tidak valid"
}

var (
	hari  = []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	bulan = []string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	tzOff = map[string]int{"WIB": 7, "WITA": 8, "WIT": 9}
)

func BuildPageData(inv *Invitation, c domain.Content, guest *GuestView, guestName string, preview bool, canonical string) PageData {
	c = c.WithDefaults()
	d := PageData{
		GiftConfirm:  c.Gift.Enabled && c.Gift.ConfirmationEnabled,
		AccessMode:   "public",
		C:            c,
		Guest:        guest,
		GuestName:    guestName,
		InvitationID: inv.ID,
		Preview:      preview,
		Theme:        "/_theme/" + inv.Theme,
		Shared:       "/_shared",
		URL:          canonical,
		OGImage:      c.Cover.Photo,
	}
	if guest != nil && guestName == "" {
		d.GuestName = guest.Name
	}

	groom := personView(c.Groom, "groom")
	bride := personView(c.Bride, "bride")
	if c.CoupleOrder == "bride_first" {
		d.Couple = []PersonView{bride, groom}
	} else {
		d.Couple = []PersonView{groom, bride}
	}
	d.Title = strings.TrimSpace(nick(d.Couple[0]) + " & " + nick(d.Couple[1]))

	now := time.Now()
	for _, e := range c.Events {
		ev := eventView(e, d.Title, now)
		d.Events = append(d.Events, ev)
	}
	for i := range d.Events {
		if !d.Events[i].Start.IsZero() {
			d.FirstEvent = &d.Events[i]
			break
		}
	}
	return d
}

func nick(p PersonView) string {
	if p.Nickname != "" {
		return p.Nickname
	}
	return p.FullName
}

func personView(p domain.Person, role string) PersonView {
	v := PersonView{Person: p, Role: role}
	if role == "groom" {
		v.RoleLabel = "Mempelai Pria"
	} else {
		v.RoleLabel = "Mempelai Wanita"
	}
	v.FatherLabel = p.Father
	if p.Father != "" && p.FatherDeceased {
		v.FatherLabel += " (Alm.)"
	}
	v.MotherLabel = p.Mother
	if p.Mother != "" && p.MotherDeceased {
		v.MotherLabel += " (Almh.)"
	}
	switch {
	case v.FatherLabel != "" && v.MotherLabel != "":
		v.ParentsLine = v.FatherLabel + " & " + v.MotherLabel
	default:
		v.ParentsLine = v.FatherLabel + v.MotherLabel
	}
	return v
}

func eventView(e domain.Event, title string, now time.Time) EventView {
	v := EventView{Event: e}
	off, ok := tzOff[e.Timezone]
	if !ok {
		off, v.Timezone = 7, "WIB"
	}
	loc := time.FixedZone(v.Timezone, off*3600)

	start := e.StartTime
	if start == "" {
		start = "00:00"
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+start, loc)
	if err != nil {
		return v
	}
	v.Start = t
	v.DayName = hari[t.Weekday()]
	v.Day = fmt.Sprintf("%02d", t.Day())
	v.MonthName = bulan[t.Month()-1]
	v.Year = fmt.Sprint(t.Year())
	v.DateLong = fmt.Sprintf("%s, %d %s %d", v.DayName, t.Day(), v.MonthName, t.Year())
	v.ISO = t.Format(time.RFC3339)

	end := "Selesai"
	endT := t.Add(2 * time.Hour)
	if e.EndTime != "" {
		if et, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+e.EndTime, loc); err == nil {
			end, endT = dotTime(e.EndTime), et
		}
	}
	if e.StartTime != "" {
		v.TimeRange = dotTime(e.StartTime) + " – " + end + " " + v.Timezone
	}
	v.IsPast = now.After(endT)

	q := url.Values{}
	q.Set("action", "TEMPLATE")
	q.Set("text", e.Name+" "+title)
	q.Set("dates", t.UTC().Format("20060102T150405Z")+"/"+endT.UTC().Format("20060102T150405Z"))
	q.Set("location", strings.TrimSpace(e.Venue+", "+e.Address))
	v.CalendarURL = "https://calendar.google.com/calendar/render?" + q.Encode()

	v.MapsLink = e.MapsURL
	if v.MapsLink == "" && (e.Venue != "" || e.Address != "") {
		v.MapsLink = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(strings.TrimSpace(e.Venue+" "+e.Address))
	}
	return v
}

func dotTime(hhmm string) string { return strings.ReplaceAll(hhmm, ":", ".") }

// GuestNameFromQuery: "?to=keluarga-dan-alumni" → "Keluarga Dan Alumni"; "Pak Joko" tetap.
func GuestNameFromQuery(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, " ") && strings.Contains(s, "-") {
		parts := strings.Split(s, "-")
		for i, p := range parts {
			if p != "" {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
		s = strings.Join(parts, " ")
	}
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80])
	}
	return s
}
