// Package service (invitation) = use case undangan, tamu, ucapan, dan halaman publik.
package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	auditport "undangan/kernel/audit"
	"undangan/services/invitation/domain"
	"undangan/kernel/security"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/kernel/authctx"
	"undangan/kernel/tenant"
)

// Details = undangan + info kuota & langganan pemilik.
type Details struct {
	Invitation            *domain.Invitation
	MaxGuests             int
	SubscriptionStatus    *string
	CheckinAvailable      bool // paket pemilik menyertakan fitur check-in
	CustomDomainAvailable bool
}

type CreateInput struct {
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	Theme     string `json:"theme"`
	Subdomain string `json:"subdomain"`
}

type InvitationService interface {
	List(ctx context.Context, actor authctx.Principal, f domain.ListFilter) ([]domain.Invitation, int, error)
	ListForUser(ctx context.Context, userID string) ([]domain.Invitation, error)
	Create(ctx context.Context, actor authctx.Principal, in CreateInput) (*Details, error)
	Get(ctx context.Context, actor authctx.Principal, id string) (*Details, error)
	UpdateContent(ctx context.Context, actor authctx.Principal, id string, c domain.Content) (*Details, error)
	Delete(ctx context.Context, actor authctx.Principal, id string) error
	SetTheme(ctx context.Context, actor authctx.Principal, id, theme string) (*Details, error)
	SetSubdomain(ctx context.Context, actor authctx.Principal, id, label string) (*Details, error)
	Publish(ctx context.Context, actor authctx.Principal, id string) (*Details, error)
	Unpublish(ctx context.Context, actor authctx.Principal, id string) (*Details, error)
	// UpdateSettings: mode akses & check-in (PIN disimpan sebagai hash).
	UpdateSettings(ctx context.Context, actor authctx.Principal, id string, in domain.SettingsInput) (*Details, error)
	CheckSubdomain(ctx context.Context, label string) (available bool, reason string, err error)

	// Load = undangan yang boleh diakses actor (customer: miliknya saja, selain itu 404).
	Load(ctx context.Context, actor authctx.Principal, id string) (*domain.Invitation, error)
	// EnsureEditable menolak perubahan oleh customer bila langganan tidak aktif (masa tenggang/habis).
	EnsureEditable(ctx context.Context, actor authctx.Principal, inv *domain.Invitation) error
	GuestLimit(ctx context.Context, inv *domain.Invitation) (int, error)

	// Port untuk modul billing (lifecycle langganan).
	SuspendForUser(ctx context.Context, userID string) (int, error)
	RestoreForUser(ctx context.Context, userID string) (int, error)
	PurgePersonalData(ctx context.Context, suspendedBefore time.Time) (int, error)
}

type invitationService struct {
	repo   domain.InvitationRepository
	subs   domain.SubscriptionReader
	themes domain.ThemeChecker
	audit  auditport.Recorder
	tx     database.TxManager
	hasher security.PasswordHasher
}

func NewInvitationService(repo domain.InvitationRepository, subs domain.SubscriptionReader, themes domain.ThemeChecker,
	audit auditport.Recorder, tx database.TxManager, hasher security.PasswordHasher) InvitationService {
	return &invitationService{repo: repo, subs: subs, themes: themes, audit: audit, tx: tx, hasher: hasher}
}

var errInvitationNotFound = apperror.NotFound("Undangan tidak ditemukan")

func (s *invitationService) Load(ctx context.Context, actor authctx.Principal, id string) (*domain.Invitation, error) {
	inv, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, errInvitationNotFound
	}
	if err != nil {
		return nil, err
	}
	if !actor.IsAdmin() && inv.UserID != actor.UserID {
		return nil, errInvitationNotFound
	}
	return inv, nil
}

func (s *invitationService) EnsureEditable(ctx context.Context, actor authctx.Principal, inv *domain.Invitation) error {
	if actor.IsAdmin() {
		return nil
	}
	status, _, _, _, ok, err := s.subs.LimitsForUser(ctx, inv.UserID)
	if err != nil {
		return err
	}
	if !ok || status != "active" {
		return &apperror.Error{Status: http.StatusForbidden, Code: "subscription_inactive",
			Message: "Masa aktif paket sudah habis. Perpanjang paket untuk mengubah undangan."}
	}
	return nil
}

func (s *invitationService) GuestLimit(ctx context.Context, inv *domain.Invitation) (int, error) {
	_, _, maxGuests, _, ok, err := s.subs.LimitsForUser(ctx, inv.UserID)
	if err != nil {
		return 0, err
	}
	if ok {
		return maxGuests, nil
	}
	return s.subs.DefaultGuestLimit(ctx)
}

func (s *invitationService) details(ctx context.Context, inv *domain.Invitation) (*Details, error) {
	d := &Details{Invitation: inv}
	status, _, maxGuests, _, ok, err := s.subs.LimitsForUser(ctx, inv.UserID)
	if err != nil {
		return nil, err
	}
	if ok {
		d.MaxGuests, d.SubscriptionStatus = maxGuests, &status
	} else if d.MaxGuests, err = s.subs.DefaultGuestLimit(ctx); err != nil {
		return nil, err
	}
	if d.CheckinAvailable, err = s.subs.CheckinAllowedForUser(ctx, inv.UserID); err != nil {
		return nil, err
	}
	if d.CustomDomainAvailable, err = s.subs.CustomDomainAllowedForUser(ctx, inv.UserID); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *invitationService) reload(ctx context.Context, id string) (*Details, error) {
	inv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.details(ctx, inv)
}

func (s *invitationService) List(ctx context.Context, actor authctx.Principal, f domain.ListFilter) ([]domain.Invitation, int, error) {
	if !actor.IsAdmin() {
		f.UserID = actor.UserID
	}
	return s.repo.List(ctx, f)
}

func (s *invitationService) ListForUser(ctx context.Context, userID string) ([]domain.Invitation, error) {
	items, _, err := s.repo.List(ctx, domain.ListFilter{UserID: userID})
	return items, err
}

func (s *invitationService) Create(ctx context.Context, actor authctx.Principal, in CreateInput) (*Details, error) {
	if in.EventType == "" {
		in.EventType = "pernikahan"
	}
	if in.EventType != "pernikahan" && in.EventType != "ngunduh_mantu" {
		return nil, apperror.Validation(map[string]string{"event_type": "Jenis acara tidak valid"})
	}

	ownerID := actor.UserID
	if actor.IsAdmin() {
		if strings.TrimSpace(in.UserID) == "" {
			return nil, apperror.Validation(map[string]string{"user_id": "Pilih customer pemilik undangan"})
		}
		ownerID = in.UserID
	} else {
		status, maxInv, _, _, ok, err := s.subs.LimitsForUser(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		if !ok || status != "active" {
			return nil, &apperror.Error{Status: http.StatusForbidden, Code: "subscription_required",
				Message: "Paket belum aktif. Pilih dan bayar paket terlebih dahulu."}
		}
		n, err := s.repo.CountByUser(ctx, ownerID)
		if err != nil {
			return nil, err
		}
		if n >= maxInv {
			return nil, apperror.Conflict("quota_exceeded", "Kuota undangan pada paket Anda sudah habis")
		}
	}

	inv := &domain.Invitation{UserID: ownerID, Status: domain.StatusDraft, Content: domain.NewContent(in.EventType)}
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.Create(ctx, inv); err != nil {
			return err
		}
		if in.Theme != "" {
			if err := s.applyTheme(ctx, actor, inv, in.Theme); err != nil {
				return err
			}
		}
		if in.Subdomain != "" {
			return s.applySubdomain(ctx, inv.ID, in.Subdomain)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.create", "invitation:"+inv.ID, map[string]any{"user_id": ownerID})
	}
	return s.reload(ctx, inv.ID)
}

func (s *invitationService) Get(ctx context.Context, actor authctx.Principal, id string) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	return s.details(ctx, inv)
}

func (s *invitationService) UpdateContent(ctx context.Context, actor authctx.Principal, id string, c domain.Content) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureEditable(ctx, actor, inv); err != nil {
		return nil, err
	}
	c, err = domain.ValidateContent(c)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateContent(ctx, id, c); err != nil {
		return nil, err
	}
	if actor.IsAdmin() && actor.UserID != inv.UserID {
		s.audit.Record(ctx, actor.UserID, "invitation.update_content", "invitation:"+id, nil)
	}
	return s.reload(ctx, id)
}

func (s *invitationService) Delete(ctx context.Context, actor authctx.Principal, id string) error {
	if !actor.IsAdmin() {
		return apperror.Forbidden("Hanya admin yang dapat menghapus undangan")
	}
	if _, err := s.Load(ctx, actor, id); err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	s.audit.Record(ctx, actor.UserID, "invitation.delete", "invitation:"+id, nil)
	return nil
}

func (s *invitationService) applyTheme(ctx context.Context, actor authctx.Principal, inv *domain.Invitation, theme string) error {
	ok, err := s.themes.IsSelectable(ctx, theme, actor.IsAdmin())
	if err != nil {
		return err
	}
	if !ok {
		return apperror.Validation(map[string]string{"theme": "Tema tidak tersedia"})
	}
	// Customer hanya boleh memilih tema satu kali. Pilihan admin juga mengunci.
	if !actor.IsAdmin() && inv.ThemeLocked() {
		if inv.Theme != nil && *inv.Theme == theme {
			return nil
		}
		return apperror.Conflict("theme_locked", "Tema sudah dipilih dan tidak bisa diganti. Hubungi admin untuk mengganti tema.")
	}
	return s.repo.SetTheme(ctx, inv.ID, theme)
}

func (s *invitationService) SetTheme(ctx context.Context, actor authctx.Principal, id, theme string) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureEditable(ctx, actor, inv); err != nil {
		return nil, err
	}
	if err := s.applyTheme(ctx, actor, inv, theme); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.set_theme", "invitation:"+id, map[string]any{"theme": theme})
	}
	return s.reload(ctx, id)
}

func (s *invitationService) CheckSubdomain(ctx context.Context, label string) (bool, string, error) {
	label = strings.ToLower(strings.TrimSpace(label))
	if !tenant.ValidSubdomain(label) {
		return false, "invalid", nil
	}
	_, err := s.repo.SubdomainOwner(ctx, label)
	switch {
	case errors.Is(err, apperror.ErrNotFound):
		return true, "", nil
	case err != nil:
		return false, "", err
	}
	return false, "taken", nil
}

func (s *invitationService) applySubdomain(ctx context.Context, invID, label string) error {
	label = strings.ToLower(strings.TrimSpace(label))
	if !tenant.ValidSubdomain(label) {
		return apperror.Validation(map[string]string{"name": "Subdomain 3–63 karakter: huruf kecil, angka, dan tanda hubung (bukan kata yang dicadangkan)"})
	}
	owner, err := s.repo.SubdomainOwner(ctx, label)
	switch {
	case err == nil && owner == invID:
		return nil
	case err == nil:
		return apperror.Conflict("subdomain_taken", "Subdomain sudah dipakai")
	case !errors.Is(err, apperror.ErrNotFound):
		return err
	}
	if err := s.repo.LockForUpdate(ctx, invID); err != nil {
		return err
	}
	return s.repo.ReplaceSubdomain(ctx, invID, label)
}

func (s *invitationService) SetSubdomain(ctx context.Context, actor authctx.Principal, id, label string) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureEditable(ctx, actor, inv); err != nil {
		return nil, err
	}
	// Link yang sudah disebar tidak boleh putus: customer tidak bisa mengganti subdomain setelah publish.
	if !actor.IsAdmin() && inv.Status == domain.StatusPublished {
		return nil, apperror.Conflict("subdomain_locked", "Subdomain tidak bisa diubah setelah undangan dipublikasikan. Batalkan publikasi atau hubungi admin.")
	}
	if err := s.tx.WithinTx(ctx, func(ctx context.Context) error { return s.applySubdomain(ctx, id, label) }); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.set_subdomain", "invitation:"+id, map[string]any{"subdomain": label})
	}
	return s.reload(ctx, id)
}

func (s *invitationService) Publish(ctx context.Context, actor authctx.Principal, id string) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureEditable(ctx, actor, inv); err != nil {
		return nil, err
	}
	_, _, _, _, hasSub, err := s.subs.LimitsForUser(ctx, inv.UserID)
	if err != nil {
		return nil, err
	}
	if missing := domain.PublishChecklist(inv, hasSub); len(missing) > 0 {
		e := apperror.Validation(missing)
		e.Code = "publish_checklist"
		return nil, e
	}
	if err := s.repo.SetStatus(ctx, id, domain.StatusPublished); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.publish", "invitation:"+id, nil)
	}
	return s.reload(ctx, id)
}

func (s *invitationService) Unpublish(ctx context.Context, actor authctx.Principal, id string) (*Details, error) {
	if _, err := s.Load(ctx, actor, id); err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, id, domain.StatusDraft); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.unpublish", "invitation:"+id, nil)
	}
	return s.reload(ctx, id)
}

func (s *invitationService) UpdateSettings(ctx context.Context, actor authctx.Principal, id string, in domain.SettingsInput) (*Details, error) {
	inv, err := s.Load(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureEditable(ctx, actor, inv); err != nil {
		return nil, err
	}
	mode, enabled := inv.AccessMode, inv.CheckinEnabled
	if in.AccessMode != nil {
		mode = *in.AccessMode
	}
	if mode != domain.AccessPublic && mode != domain.AccessGuestOnly {
		return nil, apperror.Validation(map[string]string{"access_mode": "Mode akses tidak valid"})
	}
	if in.CheckinEnabled != nil {
		enabled = *in.CheckinEnabled
	}
	var pinHash *string
	if in.CheckinPin != nil && *in.CheckinPin != "" {
		if err := domain.ValidatePin(*in.CheckinPin); err != nil {
			return nil, err
		}
		h, err := s.hasher.Hash(*in.CheckinPin)
		if err != nil {
			return nil, err
		}
		pinHash = &h
	}
	if enabled && !inv.CheckinEnabled {
		// Fitur tambahan: customer butuh paket yang menyertakan check-in. Admin boleh mengaktifkan manual.
		if !actor.IsAdmin() {
			ok, err := s.subs.CheckinAllowedForUser(ctx, inv.UserID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, &apperror.Error{Status: http.StatusForbidden, Code: "feature_unavailable",
					Message: "Fitur check-in tidak termasuk dalam paket Anda. Upgrade paket untuk mengaktifkannya."}
			}
		}
		if inv.CheckinPinHash == "" && pinHash == nil {
			return nil, apperror.Validation(map[string]string{"checkin_pin": "Buat PIN stasiun check-in (6–8 digit) untuk penerima tamu"})
		}
	}
	if err := s.repo.UpdateSettings(ctx, id, mode, enabled, pinHash); err != nil {
		return nil, err
	}
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, "invitation.update_settings", "invitation:"+id,
			map[string]any{"access_mode": mode, "checkin_enabled": enabled, "pin_changed": pinHash != nil})
	}
	return s.reload(ctx, id)
}

func (s *invitationService) SuspendForUser(ctx context.Context, userID string) (int, error) {
	return s.repo.SuspendForUser(ctx, userID)
}

func (s *invitationService) RestoreForUser(ctx context.Context, userID string) (int, error) {
	return s.repo.RestoreForUser(ctx, userID)
}

func (s *invitationService) PurgePersonalData(ctx context.Context, before time.Time) (int, error) {
	return s.repo.PurgePersonalData(ctx, before)
}
