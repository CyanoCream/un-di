// Package notify = port notifikasi keluar (Telegram, dsb). Service memakai port ini tanpa tahu salurannya.
// Implementasi wajib TIDAK pernah menggagalkan alur utama: kirim asinkron, error cukup dicatat.
package notify

import "context"

// Kind dipakai untuk memilih format & tombol di implementasi.
const (
	KindOrderCreated  = "order.created"
	KindOrderProof    = "order.proof"
	KindOrderApproved = "order.approved"
	KindOrderRejected = "order.rejected"
	KindUserRegister  = "user.registered"
)

// Action = tombol aksi (mis. Setujui / Tolak). Data dibaca kembali oleh handler bot.
type Action struct {
	Label string
	Data  string // "order:approve:<id>"
}

type Notification struct {
	Kind    string
	Title   string
	Lines   []string // baris detail: "Paket: Basic", "Nominal: Rp100.390"
	Link    string   // URL halaman admin terkait (opsional)
	Actions []Action
}

type Notifier interface {
	Notify(ctx context.Context, n Notification)
}

// Nop = notifier kosong (dipakai bila Telegram tidak dikonfigurasi).
type Nop struct{}

func (Nop) Notify(context.Context, Notification) {}
