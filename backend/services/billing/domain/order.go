package domain

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"undangan/kernel/apperror"
)

const (
	OrderAwaitingPayment      = "awaiting_payment"
	OrderAwaitingConfirmation = "awaiting_confirmation"
	OrderPaid                 = "paid"
	OrderRejected             = "rejected"
	OrderExpired              = "expired"
)

const (
	OrderTTL      = 24 * time.Hour
	MinUniqueCode = 1
	MaxUniqueCode = 499
)

// ErrOrderCodeTaken dikembalikan repository bila kode order bentrok (service mencoba kode lain).
var ErrOrderCodeTaken = errors.New("order code taken")

// wib dipakai untuk tanggal di kode order (tanpa bergantung tzdata OS).
var wib = time.FixedZone("WIB", 7*3600)

type Order struct {
	ID              string     `json:"id"`
	Code            string     `json:"code"`
	UserID          string     `json:"user_id"`
	UserName        string     `json:"user_name"`
	UserEmail       string     `json:"user_email"`
	PlanID          string     `json:"plan_id"`
	PlanName        string     `json:"plan_name"`
	Price           int64      `json:"price"`
	UniqueCode      int        `json:"unique_code"`
	Amount          int64      `json:"amount"`
	Status          string     `json:"status"`
	ProofPath       string     `json:"-"`
	ProofUploadedAt *time.Time `json:"proof_uploaded_at"`
	RejectReason    string     `json:"reject_reason"`
	ReviewedBy      *string    `json:"-"`
	ReviewedByName  *string    `json:"reviewed_by_name"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	SubscriptionID  *string    `json:"-"`
	ExpiresAt       time.Time  `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ProofURL = endpoint privat untuk melihat bukti bayar; nil bila belum ada bukti.
func (o Order) ProofURL() *string {
	if o.ProofPath == "" {
		return nil
	}
	u := "/api/v1/orders/" + o.ID + "/proof"
	return &u
}

func (o Order) MarshalJSON() ([]byte, error) {
	type alias Order
	return json.Marshal(struct {
		alias
		ProofURL *string `json:"proof_url"`
	}{alias(o), o.ProofURL()})
}

type OrderFilter struct {
	UserID string // kosong = semua (admin)
	Status string
	Query  string // nama/email user atau kode order
	Limit  int
	Offset int
}

// OrderAmount = nominal yang harus ditransfer persis (harga + kode unik).
func OrderAmount(price int64, uniqueCode int) int64 { return price + int64(uniqueCode) }

// NewOrderCode: "ORD-20260913-7KQ2" (tanggal WIB + sufiks acak).
func NewOrderCode(now time.Time, suffix string) string {
	return "ORD-" + now.In(wib).Format("20060102") + "-" + suffix
}

// PickUniqueCode memilih kode unik 1–499 sehingga price+kode tidak dipakai order terbuka lain.
// Pencarian dimulai dari start (acak dari service) lalu berputar.
func PickUniqueCode(price int64, takenAmounts []int64, start int) (int, error) {
	taken := make(map[int64]bool, len(takenAmounts))
	for _, a := range takenAmounts {
		taken[a] = true
	}
	span := MaxUniqueCode - MinUniqueCode + 1
	if start < MinUniqueCode || start > MaxUniqueCode {
		start = MinUniqueCode
	}
	for i := range span {
		code := MinUniqueCode + (start-MinUniqueCode+i)%span
		if !taken[OrderAmount(price, code)] {
			return code, nil
		}
	}
	return 0, apperror.Conflict("unique_code_exhausted", "Sedang banyak transaksi untuk paket ini, silakan coba beberapa saat lagi")
}

func NewOrder(userID string, plan *Plan, uniqueCode int, now time.Time) *Order {
	return &Order{
		UserID:     userID,
		PlanID:     plan.ID,
		PlanName:   plan.Name,
		Price:      plan.Price,
		UniqueCode: uniqueCode,
		Amount:     OrderAmount(plan.Price, uniqueCode),
		Status:     OrderAwaitingPayment,
		ExpiresAt:  now.Add(OrderTTL),
	}
}

// ErrOrderPending: customer masih punya order yang belum selesai.
func ErrOrderPending(orderID string) *apperror.Error {
	return &apperror.Error{
		Status:  http.StatusConflict,
		Code:    "order_pending",
		Message: "Anda masih memiliki order yang belum selesai",
		Fields:  map[string]string{"order_id": orderID},
	}
}

func errInvalidStatus(msg string) error { return apperror.Conflict("invalid_status", msg) }

// IsPending = order masih menunggu pembayaran/konfirmasi (memblokir order baru).
func (o *Order) IsPending() bool {
	return o.Status == OrderAwaitingPayment || o.Status == OrderAwaitingConfirmation
}

// IsOverdue = menunggu pembayaran tetapi batas waktunya sudah lewat.
func (o *Order) IsOverdue(now time.Time) bool {
	return o.Status == OrderAwaitingPayment && !now.Before(o.ExpiresAt)
}

// ---- transisi status ----

func (o *Order) CanUploadProof(now time.Time) error {
	switch {
	case o.Status == OrderRejected:
		return nil
	case o.Status == OrderAwaitingPayment && !o.IsOverdue(now):
		return nil
	case o.IsOverdue(now) || o.Status == OrderExpired:
		return apperror.Conflict("order_expired", "Batas waktu pembayaran order ini sudah lewat, silakan buat order baru")
	case o.Status == OrderAwaitingConfirmation:
		return errInvalidStatus("Bukti pembayaran sudah diunggah dan sedang dicek admin")
	default:
		return errInvalidStatus("Order ini sudah selesai")
	}
}

func (o *Order) AttachProof(path string, now time.Time) error {
	if err := o.CanUploadProof(now); err != nil {
		return err
	}
	o.ProofPath = path
	o.ProofUploadedAt = &now
	o.Status = OrderAwaitingConfirmation
	o.RejectReason = ""
	return nil
}

func (o *Order) CanApprove() error {
	if o.Status != OrderAwaitingConfirmation && o.Status != OrderAwaitingPayment {
		return errInvalidStatus("Order hanya bisa disetujui saat menunggu pembayaran atau konfirmasi")
	}
	return nil
}

func (o *Order) Approve(reviewerID, subscriptionID string, now time.Time) error {
	if err := o.CanApprove(); err != nil {
		return err
	}
	o.Status = OrderPaid
	o.ReviewedBy = &reviewerID
	o.ReviewedAt = &now
	o.SubscriptionID = &subscriptionID
	o.RejectReason = ""
	return nil
}

// NormalizeRejectReason: wajib, maks 500 karakter.
func NormalizeRejectReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	switch n := len([]rune(reason)); {
	case n == 0:
		return "", apperror.Validation(map[string]string{"reason": "Alasan penolakan wajib diisi"})
	case n > 500:
		return "", apperror.Validation(map[string]string{"reason": "Alasan penolakan maksimal 500 karakter"})
	}
	return reason, nil
}

func (o *Order) Reject(reviewerID, reason string, now time.Time) error {
	reason, err := NormalizeRejectReason(reason)
	if err != nil {
		return err
	}
	if o.Status != OrderAwaitingConfirmation {
		return errInvalidStatus("Order hanya bisa ditolak saat menunggu konfirmasi")
	}
	o.Status = OrderRejected
	o.RejectReason = reason
	o.ReviewedBy = &reviewerID
	o.ReviewedAt = &now
	return nil
}

func (o *Order) Expire(now time.Time) error {
	if !o.IsOverdue(now) {
		return errInvalidStatus("Order belum melewati batas waktu pembayaran")
	}
	o.Status = OrderExpired
	return nil
}
