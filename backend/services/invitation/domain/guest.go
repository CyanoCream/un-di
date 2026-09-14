package domain

import (
	"strconv"
	"strings"
	"time"

	"undangan/kernel/normalize"
	"undangan/kernel/apperror"
)

const (
	MaxGuestPax     = 20
	MaxImportRows   = 5000
	MaxImportBytes  = 2 << 20
	GuestCodeLength = 6
	ImportOK        = "ok"
	ImportDuplicate = "duplicate"
	ImportError     = "error"
)

type Guest struct {
	ID           string     `json:"id"`
	InvitationID string     `json:"-"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone"`
	GroupName    string     `json:"group_name"`
	Pax          int        `json:"pax"`
	Code         string     `json:"code"`
	Slug         string     `json:"slug"`
	Link         *string    `json:"link"`
	OpenedAt     *time.Time `json:"opened_at"`
	CheckedInAt  *time.Time `json:"checked_in_at"`
	CheckedInPax *int       `json:"checked_in_pax"`
	CheckedInBy  string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
}

type GuestInput struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	GroupName string `json:"group_name"`
	Pax       int    `json:"pax"`
}

// Validate merapikan & memvalidasi input tamu.
func (in GuestInput) Validate() (GuestInput, error) {
	f := apperror.Fields{}
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	switch n := len([]rune(in.Name)); {
	case n == 0:
		f.Add("name", "Nama tamu wajib diisi")
	case n > 100:
		f.Add("name", "Nama tamu maksimal 100 karakter")
	}
	raw := strings.TrimSpace(in.Phone)
	in.Phone = normalize.Phone(raw)
	if raw != "" && in.Phone == "" {
		f.Add("phone", "Nomor HP tidak valid")
	}
	in.GroupName = strings.TrimSpace(in.GroupName)
	if len([]rune(in.GroupName)) > 50 {
		f.Add("group_name", "Grup maksimal 50 karakter")
	}
	if in.Pax == 0 {
		in.Pax = 1
	}
	if in.Pax < 1 || in.Pax > MaxGuestPax {
		f.Add("pax", "Jumlah tamu 1–20 orang")
	}
	return in, f.Err()
}

// DedupKey: tamu dianggap sama bila nama (tanpa beda huruf besar/spasi) dan no HP sama.
func DedupKey(name, phone string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " ")) + "|" + phone
}

type GuestFilter struct {
	Query  string
	Group  string
	Limit  int
	Offset int
}

// ---- Import ----

// RawRow = satu baris hasil parsing file CSV/XLSX (belum divalidasi).
type RawRow struct {
	Line  int
	Name  string
	Phone string
	Group string
	Pax   string
}

type ImportRow struct {
	Row       int    `json:"row"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	GroupName string `json:"group_name"`
	Pax       int    `json:"pax"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

type ImportSummary struct {
	OK        int `json:"ok"`
	Duplicate int `json:"duplicate"`
	Error     int `json:"error"`
}

type ImportResult struct {
	Rows     []ImportRow   `json:"rows"`
	Summary  ImportSummary `json:"summary"`
	Inserted int           `json:"inserted"`
	DryRun   bool          `json:"dry_run"`
}

// EvaluateImport memvalidasi baris file terhadap tamu yang sudah ada dan sisa kuota.
// existing = DedupKey tamu yang sudah ada; remaining = sisa kuota tamu.
func EvaluateImport(raw []RawRow, existing map[string]bool, remaining int) ImportResult {
	res := ImportResult{Rows: make([]ImportRow, 0, len(raw))}
	seen := map[string]bool{}
	for _, r := range raw {
		pax := 1
		row := ImportRow{Row: r.Line, Name: strings.TrimSpace(r.Name), GroupName: strings.TrimSpace(r.Group)}
		if p := strings.TrimSpace(r.Pax); p != "" {
			v, err := strconv.Atoi(strings.TrimSuffix(p, ".0"))
			if err != nil {
				row.Status, row.Error = ImportError, "Jumlah tamu harus angka"
				row.Phone = r.Phone
				res.add(row)
				continue
			}
			pax = v
		}
		in, err := GuestInput{Name: r.Name, Phone: r.Phone, GroupName: r.Group, Pax: pax}.Validate()
		row.Name, row.Phone, row.GroupName, row.Pax = in.Name, in.Phone, in.GroupName, in.Pax
		if err != nil {
			row.Status = ImportError
			if ae, ok := err.(*apperror.Error); ok {
				row.Error = ae.Message
			}
			if row.Phone == "" {
				row.Phone = strings.TrimSpace(r.Phone)
			}
			res.add(row)
			continue
		}
		key := DedupKey(in.Name, in.Phone)
		switch {
		case existing[key] || seen[key]:
			row.Status = ImportDuplicate
		case res.Summary.OK >= remaining:
			row.Status, row.Error = ImportError, "Melebihi kuota tamu paket"
		default:
			row.Status = ImportOK
			seen[key] = true
		}
		res.add(row)
	}
	return res
}

func (r *ImportResult) add(row ImportRow) {
	r.Rows = append(r.Rows, row)
	switch row.Status {
	case ImportOK:
		r.Summary.OK++
	case ImportDuplicate:
		r.Summary.Duplicate++
	default:
		r.Summary.Error++
	}
}

// HeaderField memetakan judul kolom (berbagai variasi penulisan) ke field tamu.
func HeaderField(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(h)
	h = strings.Join(strings.Fields(h), " ")
	switch h {
	case "nama", "nama tamu", "name", "nama lengkap", "tamu":
		return "name"
	case "no hp", "hp", "nomor hp", "no whatsapp", "whatsapp", "wa", "no wa", "phone", "telepon", "no telp", "nomor telepon", "handphone":
		return "phone"
	case "grup", "group", "kategori", "keterangan", "kelompok":
		return "group"
	case "jumlah tamu", "jumlah", "pax", "jumlah orang":
		return "pax"
	}
	return ""
}
