package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"undangan/services/invitation/domain"
	"undangan/kernel/security"
	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
)

// CheckinService = stasiun penerima tamu di venue (docs/SPEC.md §9).
type CheckinService interface {
	StartSession(ctx context.Context, invID, pin, station string) (token string, expiresAt time.Time, inv *domain.Invitation, err error)
	// Authorize: token stasiun (Bearer) ATAU pemilik/admin yang login. Mengembalikan undangan & nama stasiun.
	Authorize(ctx context.Context, invID, bearer string, principal *authctx.Principal) (*domain.Invitation, string, error)
	Summary(ctx context.Context, inv *domain.Invitation) (domain.AttendanceSummary, error)
	Search(ctx context.Context, inv *domain.Invitation, q string) ([]domain.CheckinGuest, error)
	Scan(ctx context.Context, inv *domain.Invitation, input string, pax int, station, ip string) (*domain.CheckinResult, error)
	Recent(ctx context.Context, inv *domain.Invitation) ([]domain.CheckinLog, error)
}

type checkinService struct {
	invitations InvitationService
	invRepo     domain.InvitationRepository
	guests      domain.GuestRepository
	logs        domain.CheckinLogRepository
	tokens      domain.StationTokens
	hasher      security.PasswordHasher
}

func NewCheckinService(invitations InvitationService, invRepo domain.InvitationRepository, guests domain.GuestRepository,
	logs domain.CheckinLogRepository, tokens domain.StationTokens, hasher security.PasswordHasher) CheckinService {
	return &checkinService{invitations: invitations, invRepo: invRepo, guests: guests, logs: logs, tokens: tokens, hasher: hasher}
}

var (
	errStationUnauthorized = apperror.Unauthorized("station_unauthorized", "Masukkan PIN stasiun check-in")
	errCheckinDisabled     = &apperror.Error{Status: http.StatusForbidden, Code: "checkin_disabled", Message: "Fitur check-in untuk undangan ini belum diaktifkan"}
)

// pinVersion berubah setiap PIN diganti → token stasiun lama otomatis tidak berlaku.
func pinVersion(hash string) string {
	sum := sha256.Sum256([]byte(hash))
	return hex.EncodeToString(sum[:6])
}

func (s *checkinService) StartSession(ctx context.Context, invID, pin, station string) (string, time.Time, *domain.Invitation, error) {
	inv, err := s.invRepo.FindByID(ctx, invID)
	if errors.Is(err, apperror.ErrNotFound) {
		return "", time.Time{}, nil, errInvitationNotFound
	}
	if err != nil {
		return "", time.Time{}, nil, err
	}
	if !inv.CheckinEnabled || inv.CheckinPinHash == "" {
		return "", time.Time{}, nil, errCheckinDisabled
	}
	if !s.hasher.Compare(inv.CheckinPinHash, strings.TrimSpace(pin)) {
		return "", time.Time{}, nil, apperror.Unauthorized("invalid_pin", "PIN salah")
	}
	station = strings.TrimSpace(station)
	if station == "" {
		station = "Stasiun"
	}
	if r := []rune(station); len(r) > 40 {
		station = string(r[:40])
	}
	token, exp, err := s.tokens.Issue(inv.ID, station, pinVersion(inv.CheckinPinHash))
	return token, exp, inv, err
}

func (s *checkinService) Authorize(ctx context.Context, invID, bearer string, principal *authctx.Principal) (*domain.Invitation, string, error) {
	if bearer != "" {
		tokInv, station, pv, err := s.tokens.Verify(bearer)
		if err == nil && tokInv == invID {
			inv, err := s.invRepo.FindByID(ctx, invID)
			if err != nil {
				return nil, "", errStationUnauthorized
			}
			if !inv.CheckinEnabled || pinVersion(inv.CheckinPinHash) != pv {
				return nil, "", errStationUnauthorized
			}
			return inv, station, nil
		}
	}
	if principal != nil {
		inv, err := s.invitations.Load(ctx, *principal, invID)
		if err != nil {
			return nil, "", err
		}
		return inv, principal.Name, nil
	}
	return nil, "", errStationUnauthorized
}

func (s *checkinService) Summary(ctx context.Context, inv *domain.Invitation) (domain.AttendanceSummary, error) {
	return s.guests.AttendanceSummary(ctx, inv.ID)
}

func (s *checkinService) Search(ctx context.Context, inv *domain.Invitation, q string) ([]domain.CheckinGuest, error) {
	items, err := s.guests.Search(ctx, inv.ID, strings.TrimSpace(q), 20)
	if err != nil {
		return nil, err
	}
	out := make([]domain.CheckinGuest, 0, len(items))
	for i := range items {
		out = append(out, *domain.ToCheckinGuest(&items[i]))
	}
	return out, nil
}

func (s *checkinService) Scan(ctx context.Context, inv *domain.Invitation, input string, pax int, station, ip string) (*domain.CheckinResult, error) {
	logInput := input
	if r := []rune(logInput); len(r) > 200 {
		logInput = string(r[:200])
	}
	record := func(res *domain.CheckinResult, g *domain.Guest, pax *int) (*domain.CheckinResult, error) {
		l := &domain.CheckinLog{InvitationID: inv.ID, Input: logInput, Result: res.Status, Pax: pax, Station: station, IP: ip}
		if g != nil {
			l.GuestID = &g.ID
		}
		_ = s.logs.Insert(ctx, l) // log gagal tidak boleh menghentikan antrean tamu
		return res, nil
	}

	if !inv.CheckinEnabled {
		return record(&domain.CheckinResult{Status: domain.CheckinDisabled, Message: "Fitur check-in belum diaktifkan"}, nil, nil)
	}
	code := domain.ExtractGuestCode(input)
	var g *domain.Guest
	var err error
	if code != "" {
		g, err = s.guests.FindByCode(ctx, inv.ID, code)
	} else {
		err = apperror.ErrNotFound
	}
	if errors.Is(err, apperror.ErrNotFound) {
		return record(&domain.CheckinResult{Status: domain.CheckinNotFound, Message: "Tidak terdaftar sebagai tamu undangan"}, nil, nil)
	}
	if err != nil {
		return nil, err
	}

	if g.CheckedInAt != nil {
		return record(&domain.CheckinResult{Status: domain.CheckinAlready, Guest: domain.ToCheckinGuest(g),
			Message: "Sudah check-in pukul " + g.CheckedInAt.In(time.FixedZone("WIB", 7*3600)).Format("15.04") + byLabel(g.CheckedInBy)}, g, nil)
	}
	n, err := domain.ResolveCheckinPax(pax, g.Pax)
	if err != nil {
		return nil, err
	}
	updated, ok, err := s.guests.CheckIn(ctx, inv.ID, g.ID, n, station)
	if err != nil {
		return nil, err
	}
	if !ok { // kalah balapan dengan stasiun lain
		return record(&domain.CheckinResult{Status: domain.CheckinAlready, Guest: domain.ToCheckinGuest(updated),
			Message: "Sudah check-in" + byLabel(updated.CheckedInBy)}, updated, nil)
	}
	return record(&domain.CheckinResult{Status: domain.CheckinOK, Guest: domain.ToCheckinGuest(updated), Message: "Silakan masuk"}, updated, &n)
}

func byLabel(by string) string {
	if by == "" {
		return ""
	}
	return " (" + by + ")"
}

func (s *checkinService) Recent(ctx context.Context, inv *domain.Invitation) ([]domain.CheckinLog, error) {
	return s.logs.Recent(ctx, inv.ID, 20)
}
