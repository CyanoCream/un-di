package domain

import (
	"strconv"
	"strings"
	"time"

	"undangan/kernel/apperror"
)

const MaxGiftProofBytes = 5 << 20

type GiftConfirmation struct {
	ID           string     `json:"id"`
	InvitationID string     `json:"-"`
	GuestID      *string    `json:"guest_id"`
	GuestName    *string    `json:"guest_name"`
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	AccountLabel string     `json:"account_label"`
	Amount       *int64     `json:"amount"`
	Message      string     `json:"message"`
	ProofURL     string     `json:"proof_url"`
	ProofKey     string     `json:"-"`
	IP           string     `json:"-"`
	IsVerified   bool       `json:"is_verified"`
	VerifiedAt   *time.Time `json:"verified_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type GiftSummary struct {
	Count          int   `json:"count"`
	VerifiedCount  int   `json:"verified_count"`
	TotalAmount    int64 `json:"total_amount"`
	VerifiedAmount int64 `json:"verified_amount"`
}

type GiftInput struct {
	Name         string
	Type         string
	AccountLabel string
	Amount       string // dari form; boleh "Rp 250.000"
	Message      string
	GuestCode    string
}

// Validate merapikan input form konfirmasi hadiah.
func (in GiftInput) Validate() (GiftConfirmation, error) {
	f := apperror.Fields{}
	g := GiftConfirmation{
		Name:         strings.Join(strings.Fields(in.Name), " "),
		Type:         strings.TrimSpace(in.Type),
		AccountLabel: strings.TrimSpace(in.AccountLabel),
		Message:      strings.TrimSpace(in.Message),
	}
	if n := len([]rune(g.Name)); n < 2 || n > 60 {
		f.Add("name", "Nama 2–60 karakter")
	}
	if g.Type != "transfer" && g.Type != "kado" {
		f.Add("type", "Pilih jenis hadiah: transfer atau kado")
	}
	if len([]rune(g.AccountLabel)) > 120 {
		f.Add("account_label", "Tujuan terlalu panjang")
	}
	if len([]rune(g.Message)) > 500 {
		f.Add("message", "Pesan maksimal 500 karakter")
	}
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, in.Amount)
	if digits != "" && g.Type == "transfer" {
		v, err := strconv.ParseInt(digits, 10, 64)
		if err != nil || v > 1_000_000_000_000 {
			f.Add("amount", "Nominal tidak valid")
		} else {
			g.Amount = &v
		}
	}
	if g.Type == "kado" {
		g.AccountLabel = ""
	}
	return g, f.Err()
}
