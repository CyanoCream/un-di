package service

import (
	"context"
	"errors"
	"fmt"

	auditport "undangan/kernel/audit"
	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/kernel/security"
	"undangan/kernel/authctx"
)

type GuestService interface {
	List(ctx context.Context, actor authctx.Principal, invID string, f domain.GuestFilter) ([]domain.Guest, int, error)
	Create(ctx context.Context, actor authctx.Principal, invID string, in domain.GuestInput) (*domain.Guest, error)
	Update(ctx context.Context, actor authctx.Principal, invID, guestID string, in domain.GuestInput) (*domain.Guest, error)
	Delete(ctx context.Context, actor authctx.Principal, invID, guestID string) error
	// Import memproses file CSV/XLSX. dryRun=true hanya validasi (preview) tanpa menyimpan.
	Import(ctx context.Context, actor authctx.Principal, invID, filename string, data []byte, dryRun bool) (*domain.ImportResult, error)
	Export(ctx context.Context, actor authctx.Principal, invID string) (*domain.Invitation, []domain.Guest, error)
	// Check-in manual oleh pemilik/admin (dari laporan kehadiran).
	CheckIn(ctx context.Context, actor authctx.Principal, invID, guestID string, pax int) (*domain.Guest, error)
	UndoCheckIn(ctx context.Context, actor authctx.Principal, invID, guestID string) (*domain.Guest, error)
	Attendance(ctx context.Context, actor authctx.Principal, invID string, f domain.AttendanceFilter) ([]domain.AttendanceItem, int, domain.AttendanceSummary, error)
	// CanAccess = cek akses undangan tanpa mengambil tamu (untuk unduh template).
	CanAccess(ctx context.Context, actor authctx.Principal, invID string) error
}

type guestService struct {
	invitations InvitationService
	repo        domain.GuestRepository
	invRepo     domain.InvitationRepository
	parser      domain.GuestFileParser
	audit       auditport.Recorder
	tx          database.TxManager
	url         domain.URLBuilder
	logs        domain.CheckinLogRepository
}

func NewGuestService(invitations InvitationService, repo domain.GuestRepository, invRepo domain.InvitationRepository,
	parser domain.GuestFileParser, audit auditport.Recorder, tx database.TxManager, url domain.URLBuilder, logs domain.CheckinLogRepository) GuestService {
	return &guestService{invitations: invitations, repo: repo, invRepo: invRepo, parser: parser, audit: audit, tx: tx, url: url, logs: logs}
}

var errGuestNotFound = apperror.NotFound("Tamu tidak ditemukan")

func (s *guestService) withLink(inv *domain.Invitation, g *domain.Guest) {
	if base := domain.PublicURL(inv, s.url); base != nil {
		link := *base + "/" + g.Slug
		g.Link = &link
	}
}

func (s *guestService) CanAccess(ctx context.Context, actor authctx.Principal, invID string) error {
	_, err := s.invitations.Load(ctx, actor, invID)
	return err
}

func (s *guestService) List(ctx context.Context, actor authctx.Principal, invID string, f domain.GuestFilter) ([]domain.Guest, int, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, 0, err
	}
	items, total, err := s.repo.List(ctx, invID, f)
	for i := range items {
		s.withLink(inv, &items[i])
	}
	return items, total, err
}

func (s *guestService) editable(ctx context.Context, actor authctx.Principal, invID string) (*domain.Invitation, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	return inv, s.invitations.EnsureEditable(ctx, actor, inv)
}

func (s *guestService) Create(ctx context.Context, actor authctx.Principal, invID string, in domain.GuestInput) (*domain.Guest, error) {
	inv, err := s.editable(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	in, err = in.Validate()
	if err != nil {
		return nil, err
	}
	limit, err := s.invitations.GuestLimit(ctx, inv)
	if err != nil {
		return nil, err
	}
	g := &domain.Guest{InvitationID: invID, Name: in.Name, Phone: in.Phone, GroupName: in.GroupName, Pax: in.Pax}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.invRepo.LockForUpdate(ctx, invID); err != nil {
			return err
		}
		n, err := s.repo.Count(ctx, invID)
		if err != nil {
			return err
		}
		if n >= limit {
			return apperror.Conflict("quota_exceeded", fmt.Sprintf("Kuota tamu sudah penuh (maksimal %d tamu)", limit))
		}
		_, codes, err := s.repo.ExistingKeys(ctx, invID)
		if err != nil {
			return err
		}
		g.Code = uniqueCode(codes)
		slugs, err := s.repo.TakenSlugs(ctx, invID)
		if err != nil {
			return err
		}
		g.Slug = domain.UniqueSlug(g.Name, slugs)
		return s.repo.Create(ctx, g)
	})
	if err != nil {
		return nil, err
	}
	s.withLink(inv, g)
	return g, nil
}

func uniqueCode(taken map[string]bool) string {
	for {
		c := security.RandomCode(domain.GuestCodeLength)
		if !taken[c] {
			taken[c] = true
			return c
		}
	}
}

func (s *guestService) Update(ctx context.Context, actor authctx.Principal, invID, guestID string, in domain.GuestInput) (*domain.Guest, error) {
	inv, err := s.editable(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	in, err = in.Validate()
	if err != nil {
		return nil, err
	}
	g, err := s.repo.FindByID(ctx, invID, guestID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errGuestNotFound
	}
	if err != nil {
		return nil, err
	}
	g.Name, g.Phone, g.GroupName, g.Pax = in.Name, in.Phone, in.GroupName, in.Pax
	if err := s.repo.Update(ctx, g); err != nil {
		return nil, err
	}
	s.withLink(inv, g)
	return g, nil
}

func (s *guestService) Delete(ctx context.Context, actor authctx.Principal, invID, guestID string) error {
	if _, err := s.editable(ctx, actor, invID); err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, invID, guestID); err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return errGuestNotFound
		}
		return err
	}
	return nil
}

func (s *guestService) Import(ctx context.Context, actor authctx.Principal, invID, filename string, data []byte, dryRun bool) (*domain.ImportResult, error) {
	inv, err := s.editable(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	raw, err := s.parser.Parse(filename, data)
	if err != nil {
		return nil, err
	}
	limit, err := s.invitations.GuestLimit(ctx, inv)
	if err != nil {
		return nil, err
	}

	var res domain.ImportResult
	run := func(ctx context.Context) error {
		if !dryRun {
			if err := s.invRepo.LockForUpdate(ctx, invID); err != nil {
				return err
			}
		}
		keys, codes, err := s.repo.ExistingKeys(ctx, invID)
		if err != nil {
			return err
		}
		n, err := s.repo.Count(ctx, invID)
		if err != nil {
			return err
		}
		res = domain.EvaluateImport(raw, keys, max(limit-n, 0))
		res.DryRun = dryRun
		if dryRun || res.Summary.OK == 0 {
			return nil
		}
		slugs, err := s.repo.TakenSlugs(ctx, invID)
		if err != nil {
			return err
		}
		guests := make([]domain.Guest, 0, res.Summary.OK)
		for _, r := range res.Rows {
			if r.Status == domain.ImportOK {
				guests = append(guests, domain.Guest{InvitationID: invID, Name: r.Name, Phone: r.Phone, GroupName: r.GroupName, Pax: r.Pax, Code: uniqueCode(codes), Slug: domain.UniqueSlug(r.Name, slugs)})
			}
		}
		res.Inserted, err = s.repo.BulkInsert(ctx, guests)
		return err
	}
	if dryRun {
		err = run(ctx)
	} else {
		err = s.tx.WithinTx(ctx, run)
	}
	if err != nil {
		return nil, err
	}
	if !dryRun && actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "guest.import", "invitation:"+invID, map[string]any{"inserted": res.Inserted})
	}
	return &res, nil
}

func (s *guestService) CheckIn(ctx context.Context, actor authctx.Principal, invID, guestID string, pax int) (*domain.Guest, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.FindByID(ctx, invID, guestID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errGuestNotFound
	}
	if err != nil {
		return nil, err
	}
	if pax, err = domain.ResolveCheckinPax(pax, g.Pax); err != nil {
		return nil, err
	}
	g, ok, err := s.repo.CheckIn(ctx, invID, guestID, pax, "Manual: "+actor.Name)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperror.Conflict("already_checked_in", "Tamu sudah check-in sebelumnya")
	}
	_ = s.logs.Insert(ctx, &domain.CheckinLog{InvitationID: invID, GuestID: &g.ID, Input: "manual", Result: domain.CheckinOK, Pax: &pax, Station: "Manual: " + actor.Name})
	s.withLink(inv, g)
	return g, nil
}

func (s *guestService) UndoCheckIn(ctx context.Context, actor authctx.Principal, invID, guestID string) (*domain.Guest, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	g, err := s.repo.UndoCheckIn(ctx, invID, guestID)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errGuestNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = s.logs.Insert(ctx, &domain.CheckinLog{InvitationID: invID, GuestID: &g.ID, Input: "undo", Result: domain.CheckinUndo, Station: actor.Name})
	s.withLink(inv, g)
	return g, nil
}

func (s *guestService) Attendance(ctx context.Context, actor authctx.Principal, invID string, f domain.AttendanceFilter) ([]domain.AttendanceItem, int, domain.AttendanceSummary, error) {
	if _, err := s.invitations.Load(ctx, actor, invID); err != nil {
		return nil, 0, domain.AttendanceSummary{}, err
	}
	sum, err := s.repo.AttendanceSummary(ctx, invID)
	if err != nil {
		return nil, 0, sum, err
	}
	items, total, err := s.repo.Attendance(ctx, invID, f)
	return items, total, sum, err
}

func (s *guestService) Export(ctx context.Context, actor authctx.Principal, invID string) (*domain.Invitation, []domain.Guest, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.repo.ListAll(ctx, invID)
	for i := range items {
		s.withLink(inv, &items[i])
	}
	return inv, items, err
}
