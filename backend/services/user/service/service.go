// Package service (user) = use case pengelolaan akun.
package service

import (
	"context"
	"errors"

	"undangan/kernel/apperror"
	auditport "undangan/kernel/audit"
	"undangan/kernel/authctx"
	"undangan/kernel/database"
	"undangan/kernel/notify"
	"undangan/kernel/security"
	"undangan/services/user/domain"
)

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type AdminUpdateInput struct {
	Name        *string `json:"name"`
	Phone       *string `json:"phone"`
	IsSuspended *bool   `json:"is_suspended"`
}

type Service interface {
	// Register membuat akun customer (pendaftaran mandiri).
	Register(ctx context.Context, in RegisterInput) (*domain.User, error)
	// CreateCustomer oleh super admin.
	CreateCustomer(ctx context.Context, actor authctx.Principal, in RegisterInput) (*domain.User, error)
	Get(ctx context.Context, id string) (*domain.User, error)
	List(ctx context.Context, f domain.ListFilter) ([]domain.User, int, error)
	UpdateProfile(ctx context.Context, userID, name, phone string) (*domain.User, error)
	AdminUpdate(ctx context.Context, actor authctx.Principal, id string, in AdminUpdateInput) (*domain.User, error)
	ChangePassword(ctx context.Context, userID, current, next string) error
	// ResetPassword membuat password sementara dan mencabut semua sesi user.
	ResetPassword(ctx context.Context, actor authctx.Principal, id string) (string, error)
	// EnsureSuperAdmin membuat super admin pertama bila belum ada (seed dari env).
	EnsureSuperAdmin(ctx context.Context, name, email, password string) (created bool, err error)
}

type service struct {
	repo     domain.Repository
	hasher   domain.PasswordHasher
	sessions domain.SessionRevoker
	audit    auditport.Recorder
	notify   notify.Notifier
	tx       database.TxManager
}

func New(repo domain.Repository, hasher domain.PasswordHasher, sessions domain.SessionRevoker, audit auditport.Recorder,
	notifier notify.Notifier, tx database.TxManager) Service {
	if notifier == nil {
		notifier = notify.Nop{}
	}
	return &service{repo: repo, hasher: hasher, sessions: sessions, audit: audit, notify: notifier, tx: tx}
}

func (s *service) create(ctx context.Context, in RegisterInput, role string) (*domain.User, error) {
	f := apperror.Fields{}
	u := &domain.User{
		Name:  domain.ValidateName(f, in.Name),
		Email: domain.ValidateEmail(f, in.Email),
		Phone: domain.NormalizePhone(in.Phone),
		Role:  role,
	}
	domain.ValidatePassword(f, "password", in.Password)
	if err := f.Err(); err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u.PasswordHash = hash
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	if u.Role == authctx.RoleCustomer {
		lines := []string{"Nama: " + u.Name, "Email: " + u.Email}
		if u.Phone != "" {
			lines = append(lines, "WhatsApp: +"+u.Phone)
		}
		s.notify.Notify(ctx, notify.Notification{Kind: notify.KindUserRegister, Title: "Customer baru mendaftar", Lines: lines})
	}
	return u, nil
}

func (s *service) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	return s.create(ctx, in, authctx.RoleCustomer)
}

func (s *service) CreateCustomer(ctx context.Context, actor authctx.Principal, in RegisterInput) (*domain.User, error) {
	u, err := s.create(ctx, in, authctx.RoleCustomer)
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "user.create", "user:"+u.ID, map[string]any{"email": u.Email})
	return u, nil
}

func (s *service) Get(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, apperror.ErrNotFound) {
		return nil, apperror.NotFound("User tidak ditemukan")
	}
	return u, err
}

func (s *service) List(ctx context.Context, f domain.ListFilter) ([]domain.User, int, error) {
	return s.repo.List(ctx, f)
}

func (s *service) UpdateProfile(ctx context.Context, userID, name, phone string) (*domain.User, error) {
	f := apperror.Fields{}
	name = domain.ValidateName(f, name)
	if err := f.Err(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateProfile(ctx, userID, name, domain.NormalizePhone(phone)); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID)
}

func (s *service) AdminUpdate(ctx context.Context, actor authctx.Principal, id string, in AdminUpdateInput) (*domain.User, error) {
	u, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if in.Name != nil || in.Phone != nil {
			name, phone := u.Name, u.Phone
			if in.Name != nil {
				f := apperror.Fields{}
				name = domain.ValidateName(f, *in.Name)
				if err := f.Err(); err != nil {
					return err
				}
			}
			if in.Phone != nil {
				phone = domain.NormalizePhone(*in.Phone)
			}
			if err := s.repo.UpdateProfile(ctx, id, name, phone); err != nil {
				return err
			}
		}
		if in.IsSuspended != nil && *in.IsSuspended != u.IsSuspended {
			if id == actor.UserID {
				return apperror.Unprocessable("self_suspend", "Tidak bisa menonaktifkan akun sendiri")
			}
			if err := s.repo.SetSuspended(ctx, id, *in.IsSuspended); err != nil {
				return err
			}
			if *in.IsSuspended {
				if err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.audit.Record(ctx, actor.UserID, "user.update", "user:"+id, map[string]any{"name": in.Name, "phone": in.Phone, "is_suspended": in.IsSuspended})
	return s.Get(ctx, id)
}

func (s *service) ChangePassword(ctx context.Context, userID, current, next string) error {
	u, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if !s.hasher.Compare(u.PasswordHash, current) {
		return apperror.Validation(map[string]string{"current_password": "Password saat ini salah"})
	}
	f := apperror.Fields{}
	domain.ValidatePassword(f, "new_password", next)
	if err := f.Err(); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(next)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, hash)
}

func (s *service) ResetPassword(ctx context.Context, actor authctx.Principal, id string) (string, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return "", err
	}
	temp := security.RandomCode(10)
	hash, err := s.hasher.Hash(temp)
	if err != nil {
		return "", err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := s.repo.UpdatePassword(ctx, id, hash); err != nil {
			return err
		}
		return s.sessions.RevokeAllForUser(ctx, id)
	})
	if err != nil {
		return "", err
	}
	s.audit.Record(ctx, actor.UserID, "user.reset_password", "user:"+id, nil)
	return temp, nil
}

func (s *service) EnsureSuperAdmin(ctx context.Context, name, email, password string) (bool, error) {
	if email == "" || password == "" {
		return false, nil
	}
	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return false, nil
	} else if !errors.Is(err, apperror.ErrNotFound) {
		return false, err
	}
	_, err := s.create(ctx, RegisterInput{Name: name, Email: email, Password: password}, authctx.RoleSuperAdmin)
	return err == nil, err
}
