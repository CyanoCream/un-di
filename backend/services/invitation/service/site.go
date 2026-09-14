package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/tenant"
)

// SiteService = use case halaman undangan publik (dibuka tamu).
type SiteService interface {
	Resolve(ctx context.Context, host tenant.Host) (*domain.Invitation, error)
	// OpenGuest mencari tamu dari segmen path (slug nama atau kode 6 karakter) dan mencatat waktu pertama dibuka.
	OpenGuest(ctx context.Context, invID, segment string) (*domain.Guest, error)
	IsVerifiedCustomDomain(ctx context.Context, hostname string) (bool, error)
}

type siteService struct {
	invRepo domain.InvitationRepository
	guests  domain.GuestRepository
}

func NewSiteService(invRepo domain.InvitationRepository, guests domain.GuestRepository) SiteService {
	return &siteService{invRepo: invRepo, guests: guests}
}

func (s *siteService) Resolve(ctx context.Context, host tenant.Host) (*domain.Invitation, error) {
	var inv *domain.Invitation
	var err error
	switch host.Kind {
	case tenant.KindSubdomain:
		inv, err = s.invRepo.FindPublishedBySubdomain(ctx, host.Name)
	case tenant.KindCustomDomain:
		inv, err = s.invRepo.FindPublishedByCustomDomain(ctx, host.Name)
	default:
		return nil, errInvitationNotFound
	}
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errInvitationNotFound
	}
	return inv, err
}

var guestCodeRe = regexp.MustCompile(`^[A-Z0-9]{6}$`)

func (s *siteService) OpenGuest(ctx context.Context, invID, segment string) (*domain.Guest, error) {
	if len(segment) > 80 {
		return nil, errGuestNotFound
	}
	g, err := s.guests.FindBySlug(ctx, invID, strings.ToLower(segment))
	if errors.Is(err, apperror.ErrNotFound) && guestCodeRe.MatchString(strings.ToUpper(segment)) {
		g, err = s.guests.FindByCode(ctx, invID, strings.ToUpper(segment))
	}
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errGuestNotFound
	}
	if err != nil {
		return nil, err
	}
	if g.OpenedAt == nil {
		_ = s.guests.MarkOpened(ctx, g.ID) // gagal mencatat tidak boleh menggagalkan halaman
	}
	return g, nil
}

func (s *siteService) IsVerifiedCustomDomain(ctx context.Context, hostname string) (bool, error) {
	return s.invRepo.IsVerifiedCustomDomain(ctx, hostname)
}
