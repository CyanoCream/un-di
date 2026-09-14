package service

import (
	"context"
	"errors"
	"time"

	"undangan/services/billing/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
)

// UserAdapter mengimplementasikan port user/controller.SubscriptionFinder & SubscriptionGranter.
type UserAdapter struct{ subs SubscriptionService }

func NewUserAdapter(subs SubscriptionService) *UserAdapter { return &UserAdapter{subs: subs} }

// CurrentForUser mengembalikan *domain.Subscription, atau nil (untyped) bila tidak ada langganan aktif.
func (a *UserAdapter) CurrentForUser(ctx context.Context, userID string) (any, error) {
	sub, err := a.subs.Current(ctx, userID)
	if err != nil || sub == nil {
		return nil, err
	}
	return sub, nil
}

func (a *UserAdapter) GrantByAdmin(ctx context.Context, actor authctx.Principal, userID, planID string, days int) (any, error) {
	sub, err := a.subs.GrantByAdmin(ctx, actor, userID, planID, days)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// InvitationQuotaAdapter = port kuota/langganan untuk modul invitation.
type InvitationQuotaAdapter struct {
	subs  domain.SubscriptionRepository
	plans domain.PlanRepository
	now   func() time.Time
}

func NewInvitationQuotaAdapter(subs domain.SubscriptionRepository, plans domain.PlanRepository) *InvitationQuotaAdapter {
	return &InvitationQuotaAdapter{subs: subs, plans: plans, now: time.Now}
}

// LimitsForUser: ok=false bila user tidak punya langganan active/grace (dihitung dari waktu,
// sehingga tetap akurat walau job lifecycle belum berjalan). status = "active" | "grace".
func (a *InvitationQuotaAdapter) LimitsForUser(ctx context.Context, userID string) (status string, maxInvitations, maxGuests int, endsAt time.Time, ok bool, err error) {
	sub, err := a.subs.FindCurrentForUser(ctx, userID)
	if errors.Is(err, apperror.ErrNotFound) {
		return "", 0, 0, time.Time{}, false, nil
	} else if err != nil {
		return "", 0, 0, time.Time{}, false, err
	}
	st := sub.EffectiveStatus(a.now())
	if st != domain.SubscriptionActive && st != domain.SubscriptionGrace {
		return "", 0, 0, time.Time{}, false, nil
	}
	return st, sub.MaxInvitations, sub.MaxGuests, sub.EndsAt, true, nil
}

// CheckinAllowedForUser: langganan active/grace yang paketnya menyertakan fitur check-in.
func (a *InvitationQuotaAdapter) CheckinAllowedForUser(ctx context.Context, userID string) (bool, error) {
	sub, err := a.subs.FindCurrentForUser(ctx, userID)
	if errors.Is(err, apperror.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	st := sub.EffectiveStatus(a.now())
	return (st == domain.SubscriptionActive || st == domain.SubscriptionGrace) && sub.AllowCheckin, nil
}

// CustomDomainAllowedForUser: langganan active/grace yang paketnya mengizinkan domain sendiri.
func (a *InvitationQuotaAdapter) CustomDomainAllowedForUser(ctx context.Context, userID string) (bool, error) {
	sub, err := a.subs.FindCurrentForUser(ctx, userID)
	if errors.Is(err, apperror.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	st := sub.EffectiveStatus(a.now())
	if st != domain.SubscriptionActive && st != domain.SubscriptionGrace {
		return false, nil
	}
	plan, err := a.plans.FindByID(ctx, sub.PlanID)
	if err != nil {
		return false, err
	}
	return plan.AllowCustomDomain, nil
}

// DefaultGuestLimit = max_guests terkecil di antara paket aktif (fallback 100).
func (a *InvitationQuotaAdapter) DefaultGuestLimit(ctx context.Context) (int, error) {
	n, ok, err := a.plans.MinActiveMaxGuests(ctx)
	if err != nil {
		return 0, err
	}
	if !ok {
		return domain.DefaultGuestLimit, nil
	}
	return n, nil
}
