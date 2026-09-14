package service

import (
	"context"
	"log/slog"

	"undangan/services/audit/domain"
)

type Service interface {
	domain.Recorder
	List(ctx context.Context, limit, offset int) ([]domain.Log, int, error)
}

type service struct {
	repo domain.Repository
	log  *slog.Logger
}

func New(repo domain.Repository, log *slog.Logger) Service {
	return &service{repo: repo, log: log}
}

func (s *service) Record(ctx context.Context, actorID, action, target string, meta map[string]any) {
	l := &domain.Log{Action: action, Target: target, Meta: meta}
	if actorID != "" {
		l.ActorID = &actorID
	}
	if err := s.repo.Insert(ctx, l); err != nil {
		s.log.Error("audit record failed", "action", action, "target", target, "err", err)
	}
}

func (s *service) List(ctx context.Context, limit, offset int) ([]domain.Log, int, error) {
	return s.repo.List(ctx, limit, offset)
}
