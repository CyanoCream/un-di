// Package service (dashboard) = use case statistik admin.
package service

import (
	"context"

	"undangan/services/dashboard/domain"
)

type Service interface {
	Stats(ctx context.Context) (*domain.Stats, error)
}

type service struct{ repo domain.Repository }

func New(repo domain.Repository) Service { return &service{repo: repo} }

func (s *service) Stats(ctx context.Context) (*domain.Stats, error) {
	return s.repo.Stats(ctx)
}
