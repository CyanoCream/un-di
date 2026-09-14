package service

import (
	"context"
	"errors"
	"log/slog"

	auditport "undangan/kernel/audit"
	"undangan/services/theme/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
)

type Service interface {
	// Sync menyamakan tabel themes dengan folder tema. Tema baru aktif; folder hilang → nonaktif.
	Sync(ctx context.Context) error
	List(ctx context.Context, onlyActive bool) ([]domain.Theme, error)
	Update(ctx context.Context, actor authctx.Principal, slug string, in domain.UpdateInput) (*domain.Theme, error)
	// IsSelectable dipakai modul invitation: customer hanya boleh tema aktif; admin boleh tema nonaktif.
	IsSelectable(ctx context.Context, slug string, includeInactive bool) (bool, error)
}

type service struct {
	repo    domain.Repository
	catalog domain.Catalog
	audit   auditport.Recorder
	log     *slog.Logger
}

func New(repo domain.Repository, catalog domain.Catalog, audit auditport.Recorder, log *slog.Logger) Service {
	return &service{repo: repo, catalog: catalog, audit: audit, log: log}
}

func (s *service) Sync(ctx context.Context) error {
	metas, err := s.catalog.Scan()
	if err != nil {
		return err
	}
	slugs := make([]string, 0, len(metas))
	for i, m := range metas {
		if err := s.repo.Upsert(ctx, m, (i+1)*10); err != nil {
			return err
		}
		slugs = append(slugs, m.Slug)
	}
	s.log.Info("themes synced", "count", len(metas))
	return s.repo.DeactivateMissing(ctx, slugs)
}

func (s *service) List(ctx context.Context, onlyActive bool) ([]domain.Theme, error) {
	return s.repo.List(ctx, onlyActive)
}

func (s *service) Update(ctx context.Context, actor authctx.Principal, slug string, in domain.UpdateInput) (*domain.Theme, error) {
	if err := s.repo.Update(ctx, slug, in); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, apperror.NotFound("Tema tidak ditemukan")
		}
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "theme.update", "theme:"+slug, map[string]any{"is_active": in.IsActive, "is_premium": in.IsPremium, "sort_order": in.SortOrder})
	return s.repo.Get(ctx, slug)
}

func (s *service) IsSelectable(ctx context.Context, slug string, includeInactive bool) (bool, error) {
	t, err := s.repo.Get(ctx, slug)
	if errors.Is(err, apperror.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return t.IsActive || includeInactive, nil
}
