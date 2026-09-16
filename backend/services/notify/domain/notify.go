// Package domain (notify) = kontrak bot notifikasi: data ringkas order & port ke service lain.
package domain

import (
	"context"
	"strings"

	"undangan/kernel/authctx"
)

// PendingOrder = ringkasan order yang menunggu konfirmasi (untuk perintah /pending).
type PendingOrder struct {
	ID         string
	Code       string
	Customer   string
	PlanName   string
	Amount     int64
	Status     string
	HasProof   bool
	CreatedAgo string
}

// OrderOps = port ke service billing (approve/reject/daftar) untuk aksi dari Telegram.
type OrderOps interface {
	Approve(ctx context.Context, actor authctx.Principal, orderID string) (code string, err error)
	Reject(ctx context.Context, actor authctx.Principal, orderID, reason string) (code string, err error)
	Pending(ctx context.Context) ([]PendingOrder, error)
}

// AdminResolver = port ke service user: identitas super admin yang dipakai bot saat approve/reject.
type AdminResolver interface {
	SystemAdmin(ctx context.Context) (authctx.Principal, error)
}

// Callback = data tombol: "order:approve:<id>" / "order:reject:<id>".
type Callback struct {
	Entity string
	Action string
	ID     string
}

func ParseCallback(data string) (Callback, bool) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return Callback{}, false
	}
	c := Callback{Entity: parts[0], Action: parts[1], ID: parts[2]}
	if c.Entity != "order" || (c.Action != "approve" && c.Action != "reject") {
		return Callback{}, false
	}
	return c, true
}

func OrderAction(action, id string) string { return "order:" + action + ":" + id }

// RejectPrompt = teks yang dikirim bot saat meminta alasan penolakan.
// Kode order disisipkan agar balasan pengguna bisa dikaitkan kembali ke ordernya.
const RejectPrompt = "Tulis alasan penolakan untuk order "

// OrderIDFromPrompt mengambil id order dari pesan prompt yang dibalas admin.
func OrderIDFromPrompt(text string) string {
	i := strings.Index(text, "#id:")
	if i < 0 {
		return ""
	}
	rest := text[i+len("#id:"):]
	if j := strings.IndexAny(rest, " \n"); j >= 0 {
		rest = rest[:j]
	}
	return strings.TrimSpace(rest)
}
