package service

import (
	"context"
	"errors"
	"strings"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
)

type WishService interface {
	PublicList(ctx context.Context, invID string, limit, offset int) ([]domain.PublicWish, domain.WishStats, int, error)
	PublicCreate(ctx context.Context, invID string, in domain.WishInput, ip string) (*domain.PublicWish, error)
	List(ctx context.Context, actor authctx.Principal, invID string, limit, offset int) ([]domain.Wish, domain.WishStats, int, error)
	SetHidden(ctx context.Context, actor authctx.Principal, invID, wishID string, hidden bool) (*domain.Wish, error)
	Delete(ctx context.Context, actor authctx.Principal, invID, wishID string) error
}

type wishService struct {
	invitations InvitationService
	invRepo     domain.InvitationRepository
	guests      domain.GuestRepository
	repo        domain.WishRepository
}

func NewWishService(invitations InvitationService, invRepo domain.InvitationRepository, guests domain.GuestRepository, repo domain.WishRepository) WishService {
	return &wishService{invitations: invitations, invRepo: invRepo, guests: guests, repo: repo}
}

var errGuestRequired = apperror.Forbidden("Undangan ini khusus tamu terdaftar. Gunakan link undangan pribadi Anda.")

func (s *wishService) published(ctx context.Context, invID string) (*domain.Invitation, error) {
	inv, err := s.invRepo.FindPublishedByID(ctx, invID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errInvitationNotFound
	}
	return inv, err
}

func (s *wishService) PublicList(ctx context.Context, invID string, limit, offset int) ([]domain.PublicWish, domain.WishStats, int, error) {
	inv, err := s.published(ctx, invID)
	if err != nil {
		return nil, domain.WishStats{}, 0, err
	}
	stats, err := s.repo.Stats(ctx, invID, false)
	if err != nil {
		return nil, stats, 0, err
	}
	if !inv.Content.RSVP.ShowWishes {
		return []domain.PublicWish{}, stats, 0, nil
	}
	items, total, err := s.repo.List(ctx, invID, false, limit, offset)
	if err != nil {
		return nil, stats, 0, err
	}
	out := make([]domain.PublicWish, 0, len(items))
	for _, w := range items {
		out = append(out, domain.PublicWish{Name: w.Name, Attendance: w.Attendance, Pax: w.Pax, Message: w.Message, CreatedAt: w.CreatedAt})
	}
	return out, stats, total, nil
}

func (s *wishService) PublicCreate(ctx context.Context, invID string, in domain.WishInput, ip string) (*domain.PublicWish, error) {
	inv, err := s.published(ctx, invID)
	if err != nil {
		return nil, err
	}
	c := inv.Content.WithDefaults()
	if !c.RSVP.Enabled {
		return nil, apperror.Forbidden("Konfirmasi kehadiran untuk undangan ini tidak dibuka")
	}
	maxPax := c.RSVP.MaxPax
	var guestID *string
	if in.GuestCode != "" {
		if g, err := s.guests.FindByCode(ctx, invID, strings.ToUpper(in.GuestCode)); err == nil {
			guestID, maxPax = &g.ID, g.Pax
		}
	}
	if guestID == nil && inv.AccessMode == domain.AccessGuestOnly {
		return nil, errGuestRequired
	}
	in, err = in.Validate(maxPax)
	if err != nil {
		return nil, err
	}
	w := &domain.Wish{InvitationID: invID, GuestID: guestID, Name: in.Name, Attendance: in.Attendance, Pax: in.Pax, Message: in.Message, IP: ip}
	if err := s.repo.Create(ctx, w); err != nil {
		return nil, err
	}
	return &domain.PublicWish{Name: w.Name, Attendance: w.Attendance, Pax: w.Pax, Message: w.Message, CreatedAt: w.CreatedAt}, nil
}

func (s *wishService) List(ctx context.Context, actor authctx.Principal, invID string, limit, offset int) ([]domain.Wish, domain.WishStats, int, error) {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return nil, domain.WishStats{}, 0, err
	}
	stats, err := s.repo.Stats(ctx, invID, true)
	if err != nil {
		return nil, stats, 0, err
	}
	items, total, err := s.repo.List(ctx, invID, true, limit, offset)
	return items, stats, total, err
}

func (s *wishService) SetHidden(ctx context.Context, actor authctx.Principal, invID, wishID string, hidden bool) (*domain.Wish, error) {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return nil, err
	}
	w, err := s.repo.SetHidden(ctx, invID, wishID, hidden)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, apperror.NotFound("Ucapan tidak ditemukan")
	}
	return w, err
}

func (s *wishService) Delete(ctx context.Context, actor authctx.Principal, invID, wishID string) error {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return err
	}
	err := s.repo.SoftDelete(ctx, invID, wishID)
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.NotFound("Ucapan tidak ditemukan")
	}
	return err
}
