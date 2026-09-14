package service

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"time"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/storage"
	"undangan/kernel/authctx"
)

// GiftService = konfirmasi hadiah (bukti transfer / kado) dari tamu.
type GiftService interface {
	PublicCreate(ctx context.Context, invID string, in domain.GiftInput, proof []byte, ip string) (*domain.GiftConfirmation, error)
	List(ctx context.Context, actor authctx.Principal, invID string, limit, offset int) ([]domain.GiftConfirmation, int, domain.GiftSummary, error)
	Proof(ctx context.Context, actor authctx.Principal, invID, giftID string) (io.ReadSeekCloser, time.Time, error)
	SetVerified(ctx context.Context, actor authctx.Principal, invID, giftID string, verified bool) (*domain.GiftConfirmation, error)
	Delete(ctx context.Context, actor authctx.Principal, invID, giftID string) error
}

type giftService struct {
	invitations InvitationService
	invRepo     domain.InvitationRepository
	guests      domain.GuestRepository
	repo        domain.GiftRepository
	files       storage.FileStorage
}

func NewGiftService(invitations InvitationService, invRepo domain.InvitationRepository, guests domain.GuestRepository,
	repo domain.GiftRepository, files storage.FileStorage) GiftService {
	return &giftService{invitations: invitations, invRepo: invRepo, guests: guests, repo: repo, files: files}
}

var errGiftNotFound = apperror.NotFound("Konfirmasi hadiah tidak ditemukan")

func (s *giftService) PublicCreate(ctx context.Context, invID string, in domain.GiftInput, proof []byte, ip string) (*domain.GiftConfirmation, error) {
	inv, err := s.invRepo.FindPublishedByID(ctx, invID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errInvitationNotFound
	}
	if err != nil {
		return nil, err
	}
	if !inv.Content.Gift.Enabled || !inv.Content.Gift.ConfirmationEnabled {
		return nil, apperror.Forbidden("Konfirmasi hadiah tidak dibuka untuk undangan ini")
	}
	g, err := in.Validate()
	if err != nil {
		return nil, err
	}
	if in.GuestCode != "" {
		if guest, err := s.guests.FindByCode(ctx, invID, strings.ToUpper(in.GuestCode)); err == nil {
			g.GuestID = &guest.ID
		}
	}
	if g.GuestID == nil && inv.AccessMode == domain.AccessGuestOnly {
		return nil, errGuestRequired
	}
	if len(proof) == 0 {
		return nil, apperror.Validation(map[string]string{"file": "Foto bukti wajib diunggah"})
	}
	ext, err := storage.Detect(proof, storage.ImageRule)
	if err != nil {
		return nil, apperror.Validation(map[string]string{"file": "Bukti harus berupa foto JPG, PNG, atau WebP"})
	}
	key, err := s.files.SavePrivate(ctx, "gifts/"+invID, proof, ext)
	if err != nil {
		return nil, err
	}
	g.InvitationID, g.ProofKey, g.IP = invID, key, ip
	if err := s.repo.Create(ctx, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *giftService) List(ctx context.Context, actor authctx.Principal, invID string, limit, offset int) ([]domain.GiftConfirmation, int, domain.GiftSummary, error) {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return nil, 0, domain.GiftSummary{}, err
	}
	sum, err := s.repo.Summary(ctx, invID)
	if err != nil {
		return nil, 0, sum, err
	}
	items, total, err := s.repo.List(ctx, invID, limit, offset)
	return items, total, sum, err
}

func (s *giftService) find(ctx context.Context, actor authctx.Principal, invID, giftID string) (*domain.GiftConfirmation, error) {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return nil, err
	}
	g, err := s.repo.FindByID(ctx, invID, giftID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errGiftNotFound
	}
	return g, err
}

func (s *giftService) Proof(ctx context.Context, actor authctx.Principal, invID, giftID string) (io.ReadSeekCloser, time.Time, error) {
	g, err := s.find(ctx, actor, invID, giftID)
	if err != nil {
		return nil, time.Time{}, err
	}
	f, mod, err := s.files.OpenPrivate(ctx, g.ProofKey)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, time.Time{}, apperror.NotFound("File bukti tidak ditemukan")
	}
	return f, mod, err
}

func (s *giftService) SetVerified(ctx context.Context, actor authctx.Principal, invID, giftID string, verified bool) (*domain.GiftConfirmation, error) {
	if _, err := s.find(ctx, actor, invID, giftID); err != nil {
		return nil, err
	}
	return s.repo.SetVerified(ctx, invID, giftID, verified)
}

func (s *giftService) Delete(ctx context.Context, actor authctx.Principal, invID, giftID string) error {
	if _, err := s.find(ctx, actor, invID, giftID); err != nil {
		return err
	}
	return s.repo.SoftDelete(ctx, invID, giftID)
}
