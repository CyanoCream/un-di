// Package service (auth) = use case login, registrasi, refresh & logout.
package service

import (
	"context"
	"errors"
	"time"

	"undangan/services/auth/domain"
	"undangan/kernel/normalize"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/kernel/security"
	"undangan/kernel/authctx"
)

type Service interface {
	Login(ctx context.Context, email, password, portal string, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error)
	Register(ctx context.Context, in domain.RegisterInput, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error)
	// Refresh merotasi refresh token dan menerbitkan access token baru.
	Refresh(ctx context.Context, refreshToken string, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	Authenticate(accessToken string) (domain.Claims, error)
	// RevokeAllForUser memenuhi userdomain.SessionRevoker.
	RevokeAllForUser(ctx context.Context, userID string) error
}

type service struct {
	tokens     domain.TokenManager
	refresh    domain.RefreshTokenRepository
	accounts   domain.AccountStore
	hasher     security.PasswordHasher
	tx         database.TxManager
	refreshTTL time.Duration
	dummyHash  string
}

func New(tokens domain.TokenManager, refresh domain.RefreshTokenRepository, accounts domain.AccountStore,
	hasher security.PasswordHasher, tx database.TxManager, refreshTTL time.Duration) Service {
	dummy, _ := hasher.Hash("dummy-password-for-constant-time")
	return &service{tokens: tokens, refresh: refresh, accounts: accounts, hasher: hasher, tx: tx, refreshTTL: refreshTTL, dummyHash: dummy}
}

var errInvalidCredentials = apperror.Unauthorized("invalid_credentials", "Email atau password salah")
var errSessionInvalid = apperror.Unauthorized("session_invalid", "Sesi tidak valid, silakan masuk kembali")

func (s *service) Login(ctx context.Context, email, password, portal string, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error) {
	if portal != domain.PortalAdmin {
		portal = domain.PortalCustomer
	}
	u, err := s.accounts.FindByEmail(ctx, normalize.Email(email))
	if errors.Is(err, apperror.ErrNotFound) {
		s.hasher.Compare(s.dummyHash, password) // samakan waktu respons
		return nil, domain.TokenPair{}, errInvalidCredentials
	}
	if err != nil {
		return nil, domain.TokenPair{}, err
	}
	if !s.hasher.Compare(u.PasswordHash, password) {
		return nil, domain.TokenPair{}, errInvalidCredentials
	}
	if u.IsSuspended {
		return nil, domain.TokenPair{}, apperror.Forbidden("Akun dinonaktifkan. Hubungi admin.")
	}
	// Portal admin & customer terpisah: akun hanya bisa masuk ke portal sesuai role.
	switch {
	case portal == domain.PortalAdmin && u.Role != authctx.RoleSuperAdmin:
		return nil, domain.TokenPair{}, apperror.Forbidden("Akun ini bukan super admin")
	case portal == domain.PortalCustomer && u.Role != authctx.RoleCustomer:
		return nil, domain.TokenPair{}, apperror.Forbidden("Gunakan portal admin untuk akun ini")
	}
	pair, err := s.issue(ctx, u, portal, "", client)
	return u, pair, err
}

func (s *service) Register(ctx context.Context, in domain.RegisterInput, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error) {
	var u *domain.Account
	var pair domain.TokenPair
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if u, err = s.accounts.Register(ctx, in); err != nil {
			return err
		}
		pair, err = s.issue(ctx, u, domain.PortalCustomer, "", client)
		return err
	})
	return u, pair, err
}

func (s *service) Refresh(ctx context.Context, raw string, client domain.ClientInfo) (*domain.Account, domain.TokenPair, error) {
	if raw == "" {
		return nil, domain.TokenPair{}, errSessionInvalid
	}
	var u *domain.Account
	var pair domain.TokenPair
	var reuse bool
	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		t, err := s.refresh.FindByHashForUpdate(ctx, security.SHA256(raw))
		if errors.Is(err, apperror.ErrNotFound) {
			return errSessionInvalid
		}
		if err != nil {
			return err
		}
		if t.RevokedAt != nil {
			// Token lama dipakai lagi → kemungkinan dicuri. Cabut seluruh family (di luar tx yang gagal).
			reuse = true
			return errSessionInvalid
		}
		if time.Now().After(t.ExpiresAt) {
			return errSessionInvalid
		}
		if u, err = s.accounts.FindByID(ctx, t.UserID); err != nil {
			if errors.Is(err, apperror.ErrNotFound) {
				return errSessionInvalid
			}
			return err
		}
		if u.IsSuspended {
			if err := s.refresh.RevokeFamily(ctx, t.FamilyID); err != nil {
				return err
			}
			return apperror.Forbidden("Akun dinonaktifkan. Hubungi admin.")
		}
		if err := s.refresh.Revoke(ctx, t.ID); err != nil {
			return err
		}
		pair, err = s.issue(ctx, u, t.Portal, t.FamilyID, client)
		return err
	})
	if reuse {
		if t, ferr := s.refresh.FindByHashForUpdate(ctx, security.SHA256(raw)); ferr == nil {
			_ = s.refresh.RevokeFamily(ctx, t.FamilyID)
		}
	}
	return u, pair, err
}

func (s *service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		t, err := s.refresh.FindByHashForUpdate(ctx, security.SHA256(raw))
		if errors.Is(err, apperror.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return s.refresh.RevokeFamily(ctx, t.FamilyID)
	})
}

func (s *service) Authenticate(accessToken string) (domain.Claims, error) {
	return s.tokens.Verify(accessToken)
}

func (s *service) RevokeAllForUser(ctx context.Context, userID string) error {
	return s.refresh.RevokeAllForUser(ctx, userID)
}

func (s *service) issue(ctx context.Context, u *domain.Account, portal, familyID string, client domain.ClientInfo) (domain.TokenPair, error) {
	access, accessExp, err := s.tokens.Issue(domain.Claims{UserID: u.ID, Role: u.Role, Name: u.Name, Email: u.Email, Portal: portal})
	if err != nil {
		return domain.TokenPair{}, err
	}
	if familyID == "" {
		familyID = newUUID()
	}
	raw := security.RandomToken(32)
	rt := &domain.RefreshToken{
		UserID: u.ID, FamilyID: familyID, TokenHash: security.SHA256(raw), Portal: portal,
		IP: client.IP, UserAgent: truncate(client.UserAgent, 255), ExpiresAt: time.Now().Add(s.refreshTTL),
	}
	if err := s.refresh.Create(ctx, rt); err != nil {
		return domain.TokenPair{}, err
	}
	return domain.TokenPair{AccessToken: access, AccessExpiresAt: accessExp, RefreshToken: raw, RefreshExpiresAt: rt.ExpiresAt}, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
