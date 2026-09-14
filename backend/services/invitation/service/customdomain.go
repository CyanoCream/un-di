package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	auditport "undangan/kernel/audit"
	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/security"
	"undangan/kernel/authctx"
)

// CustomDomainService = hubungkan domain milik customer ke undangan.
type CustomDomainService interface {
	Set(ctx context.Context, actor authctx.Principal, invID, hostname string) (*Details, error)
	Verify(ctx context.Context, actor authctx.Principal, invID string) (*Details, error)
	Remove(ctx context.Context, actor authctx.Principal, invID string) (*Details, error)
}

type customDomainService struct {
	invitations InvitationService
	repo        domain.InvitationRepository
	subs        domain.SubscriptionReader
	resolver    domain.DNSResolver
	cfg         domain.DomainConfig
	audit       auditport.Recorder
}

func NewCustomDomainService(invitations InvitationService, repo domain.InvitationRepository, subs domain.SubscriptionReader,
	resolver domain.DNSResolver, cfg domain.DomainConfig, audit auditport.Recorder) CustomDomainService {
	return &customDomainService{invitations: invitations, repo: repo, subs: subs, resolver: resolver, cfg: cfg, audit: audit}
}

func (s *customDomainService) editable(ctx context.Context, actor authctx.Principal, invID string) (*domain.Invitation, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	return inv, s.invitations.EnsureEditable(ctx, actor, inv)
}

func (s *customDomainService) Set(ctx context.Context, actor authctx.Principal, invID, hostname string) (*Details, error) {
	inv, err := s.editable(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	if !actor.IsAdmin() {
		ok, err := s.subs.CustomDomainAllowedForUser(ctx, inv.UserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &apperror.Error{Status: http.StatusForbidden, Code: "feature_unavailable",
				Message: "Paket Anda belum mendukung domain sendiri."}
		}
	}
	host, err := domain.NormalizeHostname(hostname, s.cfg.BaseDomain)
	if err != nil {
		return nil, err
	}
	if inv.CustomDomain != nil && inv.CustomDomain.Hostname == host {
		return s.invitations.Get(ctx, actor, invID)
	}
	owner, err := s.repo.SubdomainOwner(ctx, host) // unik lintas semua hostname aktif
	switch {
	case err == nil && owner != invID:
		return nil, apperror.Conflict("domain_taken", "Domain sudah terhubung ke undangan lain")
	case err != nil && !errors.Is(err, apperror.ErrNotFound):
		return nil, err
	}
	if err := s.repo.ReplaceCustomDomain(ctx, invID, host, "undangan-verify="+strings.ToLower(security.RandomCode(20))); err != nil {
		return nil, err
	}
	s.record(ctx, actor, inv, "invitation.custom_domain.set", map[string]any{"hostname": host})
	return s.invitations.Get(ctx, actor, invID)
}

func (s *customDomainService) Verify(ctx context.Context, actor authctx.Principal, invID string) (*Details, error) {
	inv, err := s.editable(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	if inv.CustomDomain == nil {
		return nil, apperror.Unprocessable("no_custom_domain", "Belum ada domain yang ditambahkan")
	}
	if !inv.CustomDomain.Verified() {
		if missing := domain.CheckDNS(ctx, s.resolver, inv.CustomDomain, s.cfg); len(missing) > 0 {
			return nil, apperror.Unprocessable("dns_not_ready",
				"DNS belum siap: "+strings.Join(missing, "; ")+". Perubahan DNS bisa memakan waktu hingga 24 jam.")
		}
		if err := s.repo.MarkCustomDomainVerified(ctx, invID); err != nil {
			return nil, err
		}
		s.record(ctx, actor, inv, "invitation.custom_domain.verified", map[string]any{"hostname": inv.CustomDomain.Hostname})
	}
	return s.invitations.Get(ctx, actor, invID)
}

func (s *customDomainService) Remove(ctx context.Context, actor authctx.Principal, invID string) (*Details, error) {
	inv, err := s.invitations.Load(ctx, actor, invID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.RemoveCustomDomain(ctx, invID); err != nil {
		return nil, err
	}
	s.record(ctx, actor, inv, "invitation.custom_domain.remove", nil)
	return s.invitations.Get(ctx, actor, invID)
}

func (s *customDomainService) record(ctx context.Context, actor authctx.Principal, inv *domain.Invitation, action string, meta map[string]any) {
	if actor.IsAdmin() {
		s.audit.Record(ctx, actor.UserID, action, "invitation:"+inv.ID, meta)
	}
}
