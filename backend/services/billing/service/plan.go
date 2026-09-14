// Package service (billing) = use case paket, order, langganan, pengaturan pembayaran, dan job lifecycle.
package service

import (
	"context"
	"errors"

	auditport "undangan/kernel/audit"
	"undangan/services/billing/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/kernel/authctx"
)

// PlanInput dipakai untuk create (field kosong = default) dan PATCH (hanya field yang dikirim).
type PlanInput struct {
	Name              *string `json:"name"`
	Description       *string `json:"description"`
	Price             *int64  `json:"price"`
	DurationDays      *int    `json:"duration_days"`
	GraceDays         *int    `json:"grace_days"`
	MaxInvitations    *int    `json:"max_invitations"`
	MaxGuests         *int    `json:"max_guests"`
	AllowCustomDomain *bool   `json:"allow_custom_domain"`
	AllowCheckin      *bool   `json:"allow_checkin"`
	IsActive          *bool   `json:"is_active"`
	SortOrder         *int    `json:"sort_order"`
}

func (in PlanInput) applyTo(p *domain.Plan) {
	set(&p.Name, in.Name)
	set(&p.Description, in.Description)
	set(&p.Price, in.Price)
	set(&p.DurationDays, in.DurationDays)
	set(&p.GraceDays, in.GraceDays)
	set(&p.MaxInvitations, in.MaxInvitations)
	set(&p.MaxGuests, in.MaxGuests)
	set(&p.AllowCustomDomain, in.AllowCustomDomain)
	set(&p.AllowCheckin, in.AllowCheckin)
	set(&p.IsActive, in.IsActive)
	set(&p.SortOrder, in.SortOrder)
}

func set[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

type PlanService interface {
	// ListActive = paket aktif untuk customer, urut sort_order, harga.
	ListActive(ctx context.Context) ([]domain.Plan, error)
	// ListAll = semua paket (admin), termasuk yang nonaktif.
	ListAll(ctx context.Context) ([]domain.Plan, error)
	Get(ctx context.Context, id string) (*domain.Plan, error)
	Create(ctx context.Context, actor authctx.Principal, in PlanInput) (*domain.Plan, error)
	// Update parsial; paket tidak pernah dihapus, cukup is_active=false.
	Update(ctx context.Context, actor authctx.Principal, id string, in PlanInput) (*domain.Plan, error)
	// EnsureDefaultPlans mengisi paket bawaan bila tabel plans masih kosong.
	EnsureDefaultPlans(ctx context.Context) error
}

type planService struct {
	plans  domain.PlanRepository
	locker domain.Locker
	audit  auditport.Recorder
	tx     database.TxManager
}

func NewPlanService(plans domain.PlanRepository, locker domain.Locker, audit auditport.Recorder, tx database.TxManager) PlanService {
	return &planService{plans: plans, locker: locker, audit: audit, tx: tx}
}

func (s *planService) ListActive(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx, true)
}

func (s *planService) ListAll(ctx context.Context) ([]domain.Plan, error) {
	return s.plans.List(ctx, false)
}

func (s *planService) Get(ctx context.Context, id string) (*domain.Plan, error) {
	p, err := s.plans.FindByID(ctx, id)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, apperror.NotFound("Paket tidak ditemukan")
	}
	return p, err
}

func (s *planService) Create(ctx context.Context, actor authctx.Principal, in PlanInput) (*domain.Plan, error) {
	p := domain.NewPlanDefaults()
	in.applyTo(&p)
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	if err := s.plans.Create(ctx, &p); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "plan.create", "plan:"+p.ID, map[string]any{"name": p.Name, "price": p.Price})
	return &p, nil
}

func (s *planService) Update(ctx context.Context, actor authctx.Principal, id string, in PlanInput) (*domain.Plan, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	in.applyTo(p)
	if err := p.Normalize(); err != nil {
		return nil, err
	}
	if err := s.plans.Update(ctx, p); err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "plan.update", "plan:"+p.ID, map[string]any{
		"name": p.Name, "price": p.Price, "is_active": p.IsActive,
	})
	return p, nil
}

func (s *planService) EnsureDefaultPlans(ctx context.Context) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.locker.LockTx(ctx, "billing:plan_seed"); err != nil {
			return err
		}
		n, err := s.plans.Count(ctx)
		if err != nil || n > 0 {
			return err
		}
		for _, p := range domain.DefaultPlans() {
			if err := s.plans.Create(ctx, &p); err != nil {
				return err
			}
		}
		return nil
	})
}
