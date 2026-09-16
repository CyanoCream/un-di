package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"undangan/kernel/database"
	"undangan/services/billing/domain"
)

// PersonalDataRetention = data tamu & ucapan undangan yang ter-suspend selama ini dihapus permanen (UU PDP).
const PersonalDataRetention = 365 * 24 * time.Hour

const lifecycleLockKey = "billing:lifecycle"

// Lifecycle = job berkala: order kedaluwarsa, pengingat, masa tenggang, expired, purge data pribadi.
// Setiap langkah idempotent; advisory lock mencegah instance lain berjalan bersamaan.
type Lifecycle struct {
	orders        domain.OrderRepository
	subs          domain.SubscriptionRepository
	notifications domain.NotificationRepository
	invitations   domain.InvitationLifecycle
	locker        domain.Locker
	tx            database.TxManager
	log           *slog.Logger
	now           func() time.Time
}

// NewLifecycle: invitations boleh nil (suspend/purge undangan dilewati).
func NewLifecycle(orders domain.OrderRepository, subs domain.SubscriptionRepository, notifications domain.NotificationRepository,
	invitations domain.InvitationLifecycle, locker domain.Locker, tx database.TxManager, log *slog.Logger) *Lifecycle {
	return &Lifecycle{orders: orders, subs: subs, notifications: notifications, invitations: invitations,
		locker: locker, tx: tx, log: log, now: time.Now}
}

// Start menjalankan RunOnce segera, lalu setiap `every` sampai ctx selesai.
func (l *Lifecycle) Start(ctx context.Context, every time.Duration) {
	run := func() {
		if err := l.RunOnce(ctx); err != nil && ctx.Err() == nil {
			l.log.Error("billing lifecycle failed", "err", err)
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

func (l *Lifecycle) RunOnce(ctx context.Context) error {
	release, ok, err := l.locker.TryLock(ctx, lifecycleLockKey)
	if err != nil {
		return fmt.Errorf("lifecycle lock: %w", err)
	}
	if !ok {
		l.log.Info("billing lifecycle skipped: dijalankan instance lain")
		return nil
	}
	defer release()

	now := l.now()
	steps := []struct {
		name string
		fn   func(context.Context, time.Time) (int, error)
	}{
		{"expire_orders", l.expireOrders},
		{"reminders", l.sendReminders},
		{"enter_grace", l.enterGrace},
		{"expire_subscriptions", l.expireSubscriptions},
		{"purge_personal_data", l.purgePersonalData},
	}
	var firstErr error
	attrs := []any{}
	for _, st := range steps {
		n, err := st.fn(ctx, now)
		if err != nil {
			l.log.Error("billing lifecycle step failed", "step", st.name, "err", err)
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", st.name, err)
			}
		}
		attrs = append(attrs, st.name, n)
	}
	l.log.Info("billing lifecycle", attrs...)
	return firstErr
}

// (1) awaiting_payment lewat expires_at → expired.
func (l *Lifecycle) expireOrders(ctx context.Context, now time.Time) (int, error) {
	return l.orders.ExpireOverdue(ctx, now)
}

// (2) pengingat 7/3/1 hari sebelum langganan aktif berakhir.
func (l *Lifecycle) sendReminders(ctx context.Context, now time.Time) (int, error) {
	subs, err := l.subs.ListByStatusEndingBefore(ctx, domain.SubscriptionActive, now.Add(domain.ReminderWindow))
	if err != nil {
		return 0, err
	}
	sent := 0
	for _, s := range subs {
		kind, ok := domain.ReminderKind(now, s.EndsAt)
		if !ok {
			continue
		}
		created, err := l.notifications.Insert(ctx, domain.Notification{
			UserID: s.UserID, SubscriptionID: s.ID, Kind: kind, Channel: domain.ChannelInApp, Ref: domain.NotificationRef(s.EndsAt),
		})
		if err != nil {
			return sent, err
		}
		if created {
			sent++
			// TODO: kirim email / WhatsApp pengingat perpanjangan.
			l.log.Info("billing reminder queued (email/WA belum diimplementasikan)",
				"kind", kind, "user_id", s.UserID, "subscription_id", s.ID, "ends_at", s.EndsAt)
		}
	}
	return sent, nil
}

// (3) active dengan ends_at lewat → grace.
func (l *Lifecycle) enterGrace(ctx context.Context, now time.Time) (int, error) {
	subs, err := l.subs.ListByStatusEndingBefore(ctx, domain.SubscriptionActive, now)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, s := range subs {
		changed := false
		err := l.tx.WithinTx(ctx, func(ctx context.Context) error {
			var err error
			changed, err = l.subs.TransitionStatus(ctx, s.ID, domain.SubscriptionActive, domain.SubscriptionGrace)
			if err != nil || !changed {
				return err
			}
			_, err = l.notifications.Insert(ctx, domain.Notification{
				UserID: s.UserID, SubscriptionID: s.ID, Kind: domain.NotifyGrace, Channel: domain.ChannelInApp, Ref: domain.NotificationRef(s.EndsAt),
			})
			if err == nil {
				// TODO: kirim email / WhatsApp pemberitahuan masa tenggang.
				l.log.Info("billing subscription entered grace", "user_id", s.UserID, "subscription_id", s.ID, "grace_ends_at", s.GraceEndsAt)
			}
			return err
		})
		if err != nil {
			return n, err
		}
		if changed {
			n++
		}
	}
	return n, nil
}

// (4) grace dengan grace_ends_at lewat → expired + suspend undangan (bila tidak ada langganan hidup lain).
func (l *Lifecycle) expireSubscriptions(ctx context.Context, now time.Time) (int, error) {
	subs, err := l.subs.ListByStatusEndingBefore(ctx, domain.SubscriptionGrace, now)
	if err != nil {
		return 0, err
	}
	n := 0
	var firstErr error
	for _, s := range subs {
		changed := false
		err := l.tx.WithinTx(ctx, func(ctx context.Context) error {
			var err error
			changed, err = l.subs.TransitionStatus(ctx, s.ID, domain.SubscriptionGrace, domain.SubscriptionExpired)
			if err != nil || !changed {
				return err
			}
			if _, err := l.notifications.Insert(ctx, domain.Notification{
				UserID: s.UserID, SubscriptionID: s.ID, Kind: domain.NotifyExpired, Channel: domain.ChannelInApp, Ref: domain.NotificationRef(s.GraceEndsAt),
			}); err != nil {
				return err
			}
			live, err := l.subs.HasLive(ctx, s.UserID)
			if err != nil || live || l.invitations == nil {
				return err
			}
			suspended, err := l.invitations.SuspendForUser(ctx, s.UserID)
			if err == nil && suspended > 0 {
				l.log.Info("billing invitations suspended", "user_id", s.UserID, "count", suspended)
			}
			return err
		})
		if err != nil {
			// Satu user gagal tidak menghentikan user lain; dicoba lagi pada putaran berikutnya.
			l.log.Error("billing expire subscription failed", "subscription_id", s.ID, "err", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if changed {
			n++
		}
	}
	return n, firstErr
}

// (5) hapus permanen data pribadi undangan yang sudah lama ter-suspend.
func (l *Lifecycle) purgePersonalData(ctx context.Context, now time.Time) (int, error) {
	if l.invitations == nil {
		return 0, nil
	}
	return l.invitations.PurgePersonalData(ctx, now.Add(-PersonalDataRetention))
}
