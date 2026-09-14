// Package domain (billing) = paket, order pembayaran manual, langganan, pengaturan pembayaran,
// aturan bisnis murni, serta kontrak repository & port lintas modul.
package domain

import (
	"strings"
	"time"

	"undangan/kernel/apperror"
)

type Plan struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Price             int64     `json:"price"`
	DurationDays      int       `json:"duration_days"`
	GraceDays         int       `json:"grace_days"`
	MaxInvitations    int       `json:"max_invitations"`
	MaxGuests         int       `json:"max_guests"`
	AllowCustomDomain bool      `json:"allow_custom_domain"`
	AllowCheckin      bool      `json:"allow_checkin"` // fitur tambahan: check-in QR di venue
	IsActive          bool      `json:"is_active"`
	SortOrder         int       `json:"sort_order"`
	CreatedAt         time.Time `json:"-"`
}

// DefaultGuestLimit dipakai bila tidak ada paket aktif sama sekali.
const DefaultGuestLimit = 100

// NewPlanDefaults = nilai awal paket baru (sama dengan default kolom tabel plans).
func NewPlanDefaults() Plan {
	return Plan{DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: DefaultGuestLimit, IsActive: true}
}

// DefaultPlans = paket bawaan saat tabel plans masih kosong.
func DefaultPlans() []Plan {
	return []Plan{
		{
			Name: "Paket Basic", Description: "1 undangan digital, hingga 100 tamu, bisa pakai domain sendiri, aktif 30 hari",
			Price: 100_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 100,
			AllowCustomDomain: true, IsActive: true, SortOrder: 1,
		},
		{
			Name: "Paket Premium", Description: "1 undangan digital, hingga 500 tamu, check-in QR di venue, custom domain, aktif 30 hari",
			Price: 175_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 500,
			AllowCustomDomain: true, AllowCheckin: true, IsActive: true, SortOrder: 2,
		},
	}
}

// Normalize merapikan field teks lalu memvalidasi paket.
func (p *Plan) Normalize() error {
	f := apperror.Fields{}
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	switch n := len([]rune(p.Name)); {
	case n < 2:
		f.Add("name", "Nama paket minimal 2 karakter")
	case n > 100:
		f.Add("name", "Nama paket maksimal 100 karakter")
	}
	if len([]rune(p.Description)) > 500 {
		f.Add("description", "Deskripsi maksimal 500 karakter")
	}
	if p.Price < 0 {
		f.Add("price", "Harga tidak boleh negatif")
	}
	if p.DurationDays <= 0 {
		f.Add("duration_days", "Durasi minimal 1 hari")
	} else if p.DurationDays > MaxGrantDays {
		f.Add("duration_days", "Durasi maksimal 3650 hari")
	}
	if p.GraceDays < 0 {
		f.Add("grace_days", "Masa tenggang tidak boleh negatif")
	} else if p.GraceDays > 365 {
		f.Add("grace_days", "Masa tenggang maksimal 365 hari")
	}
	if p.MaxInvitations < 1 {
		f.Add("max_invitations", "Kuota undangan minimal 1")
	}
	if p.MaxGuests < 1 {
		f.Add("max_guests", "Kuota tamu minimal 1")
	}
	return f.Err()
}
