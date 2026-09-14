package domain

import (
	"context"
	"time"
)

// PaymentSettingsKey = key baris di tabel settings.
const PaymentSettingsKey = "payment"

type PaymentSettings struct {
	QRISImage     string `json:"qris_image"`
	MerchantName  string `json:"merchant_name"`
	Instructions  string `json:"instructions"`
	AdminWhatsApp string `json:"admin_whatsapp"` // 628xx
}

type SubscriptionFilter struct {
	Status string
	Query  string // nama/email user
	Limit  int
	Offset int
}

// ---- repository ----

type PlanRepository interface {
	Create(ctx context.Context, p *Plan) error
	Update(ctx context.Context, p *Plan) error
	FindByID(ctx context.Context, id string) (*Plan, error)
	// List urut sort_order, price.
	List(ctx context.Context, activeOnly bool) ([]Plan, error)
	Count(ctx context.Context) (int, error)
	// MinActiveMaxGuests: ok=false bila tidak ada paket aktif.
	MinActiveMaxGuests(ctx context.Context) (n int, ok bool, err error)
}

type SubscriptionRepository interface {
	Create(ctx context.Context, s *Subscription) error
	// Update menyimpan plan_id, status, periode, dan snapshot kuota.
	Update(ctx context.Context, s *Subscription) error
	FindByID(ctx context.Context, id string) (*Subscription, error)
	// FindCurrentForUser = langganan active/grace terbaru; ErrNotFound bila tidak ada.
	FindCurrentForUser(ctx context.Context, userID string) (*Subscription, error)
	ListByUser(ctx context.Context, userID string) ([]Subscription, error)
	List(ctx context.Context, f SubscriptionFilter) ([]Subscription, int, error)
	HasLive(ctx context.Context, userID string) (bool, error)
	// ListByStatusEndingBefore: status active → ends_at < t; status grace → grace_ends_at < t.
	ListByStatusEndingBefore(ctx context.Context, status string, t time.Time) ([]Subscription, error)
	// TransitionStatus mengubah status hanya bila status saat ini = from.
	TransitionStatus(ctx context.Context, id, from, to string) (bool, error)
}

type OrderRepository interface {
	// Create mengembalikan ErrOrderCodeTaken bila kode sudah dipakai (tanpa membatalkan transaksi).
	Create(ctx context.Context, o *Order) error
	// Update menyimpan status, bukti, alasan tolak, reviewer, subscription_id.
	Update(ctx context.Context, o *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
	// FindByIDForUpdate mengunci baris order (wajib di dalam transaksi).
	FindByIDForUpdate(ctx context.Context, id string) (*Order, error)
	List(ctx context.Context, f OrderFilter) ([]Order, int, error)
	// ListPendingByUser = order awaiting_payment/awaiting_confirmation milik user.
	ListPendingByUser(ctx context.Context, userID string) ([]Order, error)
	// TakenAmounts = amount order terbuka (menunggu konfirmasi, atau menunggu pembayaran & belum lewat batas) di rentang [min,max].
	TakenAmounts(ctx context.Context, lo, hi int64, now time.Time) ([]int64, error)
	// ExpireOverdue menandai awaiting_payment yang lewat expires_at sebagai expired.
	ExpireOverdue(ctx context.Context, now time.Time) (int, error)
}

type SettingsRepository interface {
	// GetPayment mengembalikan ErrNotFound bila belum pernah disimpan.
	GetPayment(ctx context.Context) (*PaymentSettings, error)
	SavePayment(ctx context.Context, s *PaymentSettings) error
}

type NotificationRepository interface {
	// Insert idempotent (ON CONFLICT DO NOTHING); created=false bila sudah pernah dibuat.
	Insert(ctx context.Context, n Notification) (created bool, err error)
}

// Locker = advisory lock Postgres.
type Locker interface {
	// LockTx menunggu lock sampai transaksi selesai; wajib dipanggil di dalam TxManager.WithinTx.
	LockTx(ctx context.Context, key string) error
	// TryLock lock sesi tanpa menunggu; ok=false bila dipegang proses lain. release wajib dipanggil bila ok.
	TryLock(ctx context.Context, key string) (release func(), ok bool, err error)
}

// ---- port lintas modul ----

// InvitationLifecycle diimplementasikan modul invitation.
type InvitationLifecycle interface {
	// SuspendForUser: langganan expired → suspend + soft-delete undangan, lepas subdomain.
	SuspendForUser(ctx context.Context, userID string) (int, error)
	// RestoreForUser: langganan aktif lagi → pulihkan yang dilakukan SuspendForUser.
	RestoreForUser(ctx context.Context, userID string) (int, error)
	// PurgePersonalData: hapus permanen tamu & ucapan undangan yang lama ter-suspend (UU PDP).
	PurgePersonalData(ctx context.Context, suspendedBefore time.Time) (int, error)
}
