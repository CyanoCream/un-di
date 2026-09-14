package domain

import (
	"time"

	"undangan/kernel/apperror"
)

const (
	SubscriptionActive    = "active"
	SubscriptionGrace     = "grace"
	SubscriptionExpired   = "expired"
	SubscriptionCancelled = "cancelled"
)

// MaxGrantDays = batas atas hari yang boleh diberikan sekali grant (10 tahun).
const MaxGrantDays = 3650

type Subscription struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	UserName       string    `json:"user_name"`
	UserEmail      string    `json:"user_email"`
	PlanID         string    `json:"plan_id"`
	PlanName       string    `json:"plan_name"`
	Status         string    `json:"status"`
	StartsAt       time.Time `json:"starts_at"`
	EndsAt         time.Time `json:"ends_at"`
	GraceEndsAt    time.Time `json:"grace_ends_at"`
	MaxInvitations int       `json:"max_invitations"`
	MaxGuests      int       `json:"max_guests"`
	AllowCheckin   bool      `json:"allow_checkin"`
	CreatedAt      time.Time `json:"created_at"`
}

// IsLive = langganan masih memberi akses (aktif atau masa tenggang).
func (s *Subscription) IsLive() bool {
	return s != nil && (s.Status == SubscriptionActive || s.Status == SubscriptionGrace)
}

// EffectiveStatus = status berdasarkan waktu, walau job lifecycle belum sempat berjalan.
func (s *Subscription) EffectiveStatus(now time.Time) string {
	if !s.IsLive() {
		return s.Status
	}
	switch {
	case now.After(s.GraceEndsAt):
		return SubscriptionExpired
	case now.After(s.EndsAt):
		return SubscriptionGrace
	default:
		return s.Status
	}
}

// ValidateGrantDays: 0 = pakai durasi paket.
func ValidateGrantDays(days int) error {
	if days < 0 || days > MaxGrantDays {
		return apperror.Validation(map[string]string{"days": "Jumlah hari harus antara 1 dan 3650"})
	}
	return nil
}

// ExtendPeriod menghitung periode langganan setelah grant/perpanjangan.
//   - current aktif/tenggang → mulai tetap, ends_at = max(now, ends_at) + days
//   - selain itu → langganan baru mulai sekarang
//
// days <= 0 → plan.DurationDays. grace_ends_at = ends_at + plan.GraceDays.
func ExtendPeriod(now time.Time, current *Subscription, plan *Plan, days int) (start, end, graceEnd time.Time) {
	if days <= 0 {
		days = plan.DurationDays
	}
	period := time.Duration(days) * 24 * time.Hour
	if current.IsLive() {
		start = current.StartsAt
		base := current.EndsAt
		if now.After(base) {
			base = now
		}
		end = base.Add(period)
	} else {
		start = now
		end = now.Add(period)
	}
	graceEnd = end.Add(time.Duration(plan.GraceDays) * 24 * time.Hour)
	return start, end, graceEnd
}

// ApplyGrant memperbarui langganan aktif/tenggang (current != nil) atau membuat entity baru.
func ApplyGrant(now time.Time, current *Subscription, userID string, plan *Plan, days int) *Subscription {
	start, end, grace := ExtendPeriod(now, current, plan, days)
	s := &Subscription{UserID: userID}
	if current.IsLive() {
		cp := *current
		s = &cp
	}
	s.PlanID = plan.ID
	s.PlanName = plan.Name
	s.Status = SubscriptionActive
	s.StartsAt, s.EndsAt, s.GraceEndsAt = start, end, grace
	s.MaxInvitations = plan.MaxInvitations
	s.MaxGuests = plan.MaxGuests
	s.AllowCheckin = plan.AllowCheckin
	return s
}

// ---- notifikasi lifecycle ----

const (
	NotifyReminder7d = "reminder_7d"
	NotifyReminder3d = "reminder_3d"
	NotifyReminder1d = "reminder_1d"
	NotifyGrace      = "grace"
	NotifyExpired    = "expired"

	ChannelInApp = "in_app"
)

// ReminderWindow = pengingat paling awal dikirim 7 hari sebelum berakhir.
const ReminderWindow = 7 * 24 * time.Hour

// ReminderKind memilih pengingat paling sempit yang berlaku (≤1 hari, ≤3 hari, ≤7 hari).
// ok=false bila masih > 7 hari atau sudah lewat.
func ReminderKind(now, endsAt time.Time) (string, bool) {
	left := endsAt.Sub(now)
	switch {
	case left <= 0:
		return "", false
	case left <= 24*time.Hour:
		return NotifyReminder1d, true
	case left <= 3*24*time.Hour:
		return NotifyReminder3d, true
	case left <= ReminderWindow:
		return NotifyReminder7d, true
	default:
		return "", false
	}
}

// NotificationRef mengikat notifikasi ke satu periode langganan, supaya setelah diperpanjang
// pengingat periode berikutnya tetap terkirim (unik: subscription_id, kind, channel, ref).
func NotificationRef(periodEnd time.Time) string {
	return periodEnd.UTC().Truncate(time.Second).Format(time.RFC3339)
}

type Notification struct {
	UserID         string
	SubscriptionID string
	Kind           string
	Channel        string
	Ref            string
}
