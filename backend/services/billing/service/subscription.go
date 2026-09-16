package service

import (
	"context"
	"errors"
	"time"

	"undangan/kernel/apperror"
	auditport "undangan/kernel/audit"
	"undangan/kernel/authctx"
	"undangan/kernel/database"
	"undangan/services/billing/domain"
)

// MySubscription = respons GET /me/subscription.
type MySubscription struct {
	Current *domain.Subscription  `json:"current"`
	History []domain.Subscription `json:"history"`
}

type SubscriptionService interface {
	// Grant memberi/memperpanjang langganan (days 0 = durasi paket) lalu memulihkan undangan user.
	// Ikut transaksi pemanggil bila ada (dipakai approve order).
	Grant(ctx context.Context, userID, planID string, days int) (*domain.Subscription, error)
	// GrantByAdmin = Grant manual oleh super admin + audit subscription.grant.
	GrantByAdmin(ctx context.Context, actor authctx.Principal, userID, planID string, days int) (*domain.Subscription, error)
	// Current = langganan active/grace terbaru; nil bila tidak ada.
	Current(ctx context.Context, userID string) (*domain.Subscription, error)
	ForUser(ctx context.Context, userID string) (*MySubscription, error)
	List(ctx context.Context, f domain.SubscriptionFilter) ([]domain.Subscription, int, error)
}

type subscriptionService struct {
	subs        domain.SubscriptionRepository
	plans       domain.PlanRepository
	invitations domain.InvitationLifecycle
	locker      domain.Locker
	audit       auditport.Recorder
	tx          database.TxManager
	now         func() time.Time
}

// NewSubscriptionService: invitations boleh nil (lifecycle undangan dilewati).
func NewSubscriptionService(subs domain.SubscriptionRepository, plans domain.PlanRepository, invitations domain.InvitationLifecycle,
	locker domain.Locker, audit auditport.Recorder, tx database.TxManager) SubscriptionService {
	return &subscriptionService{subs: subs, plans: plans, invitations: invitations, locker: locker, audit: audit, tx: tx, now: time.Now}
}

func (s *subscriptionService) Grant(ctx context.Context, userID, planID string, days int) (*domain.Subscription, error) {
	if err := domain.ValidateGrantDays(days); err != nil {
		return nil, err
	}
	var id string
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		// Serialisasi grant per user supaya tidak ada dua langganan aktif paralel.
		if err := s.locker.LockTx(ctx, "billing:subscription:"+userID); err != nil {
			return err
		}
		plan, err := s.plans.FindByID(ctx, planID)
		if errors.Is(err, apperror.ErrNotFound) {
			return apperror.Validation(map[string]string{"plan_id": "Paket tidak ditemukan"})
		} else if err != nil {
			return err
		}
		current, err := s.subs.FindCurrentForUser(ctx, userID)
		if errors.Is(err, apperror.ErrNotFound) {
			current = nil
		} else if err != nil {
			return err
		}

		next := domain.ApplyGrant(s.now(), current, userID, plan, days)
		if next.ID != "" {
			err = s.subs.Update(ctx, next)
		} else {
			err = s.subs.Create(ctx, next)
		}
		if err != nil {
			return err
		}
		id = next.ID
		if s.invitations != nil {
			if _, err := s.invitations.RestoreForUser(ctx, userID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.subs.FindByID(ctx, id)
}

func (s *subscriptionService) GrantByAdmin(ctx context.Context, actor authctx.Principal, userID, planID string, days int) (*domain.Subscription, error) {
	sub, err := s.Grant(ctx, userID, planID, days)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "subscription.grant", "user:"+userID, map[string]any{
		"subscription_id": sub.ID, "plan_id": planID, "days": days, "ends_at": sub.EndsAt,
	})
	return sub, nil
}

func (s *subscriptionService) Current(ctx context.Context, userID string) (*domain.Subscription, error) {
	sub, err := s.subs.FindCurrentForUser(ctx, userID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, nil
	}
	return sub, err
}

func (s *subscriptionService) ForUser(ctx context.Context, userID string) (*MySubscription, error) {
	current, err := s.Current(ctx, userID)
	if err != nil {
		return nil, err
	}
	history, err := s.subs.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if history == nil {
		history = []domain.Subscription{}
	}
	return &MySubscription{Current: current, History: history}, nil
}

func (s *subscriptionService) List(ctx context.Context, f domain.SubscriptionFilter) ([]domain.Subscription, int, error) {
	return s.subs.List(ctx, f)
}
