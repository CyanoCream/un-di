package domain

import (
	"strings"
	"time"

	"undangan/kernel/apperror"
)

type Wish struct {
	ID           string    `json:"id"`
	InvitationID string    `json:"-"`
	GuestID      *string   `json:"guest_id"`
	Name         string    `json:"name"`
	Attendance   string    `json:"attendance"`
	Pax          int       `json:"pax"`
	Message      string    `json:"message"`
	IP           string    `json:"-"`
	IsHidden     bool      `json:"is_hidden"`
	CreatedAt    time.Time `json:"created_at"`
}

// PublicWish = bentuk ucapan yang boleh dilihat tamu (tanpa id/guest/ip).
type PublicWish struct {
	Name       string    `json:"name"`
	Attendance string    `json:"attendance"`
	Pax        int       `json:"pax"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

type WishStats struct {
	Hadir    int `json:"hadir"`
	Tidak    int `json:"tidak"`
	Ragu     int `json:"ragu"`
	TotalPax int `json:"total_pax"`
}

type WishInput struct {
	Name       string `json:"name"`
	Attendance string `json:"attendance"`
	Pax        int    `json:"pax"`
	Message    string `json:"message"`
	GuestCode  string `json:"guest_code"`
}

// Validate: maxPax dari pengaturan RSVP undangan (atau pax tamu bila link personal).
func (in WishInput) Validate(maxPax int) (WishInput, error) {
	f := apperror.Fields{}
	in.Name = strings.Join(strings.Fields(in.Name), " ")
	if n := len([]rune(in.Name)); n < 2 || n > 60 {
		f.Add("name", "Nama 2–60 karakter")
	}
	switch in.Attendance {
	case "hadir", "tidak", "ragu":
	default:
		f.Add("attendance", "Pilih konfirmasi kehadiran")
	}
	if in.Attendance != "hadir" {
		in.Pax = 0
	} else {
		if in.Pax < 1 {
			in.Pax = 1
		}
		if in.Pax > maxPax {
			f.Add("pax", "Jumlah tamu melebihi batas undangan")
		}
	}
	in.Message = strings.TrimSpace(in.Message)
	if len([]rune(in.Message)) > 1000 {
		f.Add("message", "Ucapan maksimal 1000 karakter")
	}
	return in, f.Err()
}
