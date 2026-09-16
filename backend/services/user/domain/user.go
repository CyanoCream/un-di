// Package domain (user) = entity User, aturan validasi, dan kontrak repository.
package domain

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
	"undangan/kernel/normalize"
	"undangan/kernel/security"
)

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	IsSuspended  bool      `json:"is_suspended"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

func (u *User) IsAdmin() bool { return u.Role == authctx.RoleSuperAdmin }

type ListFilter struct {
	Query  string
	Role   string
	Limit  int
	Offset int
}

// Repository = kontrak persistence User (implementasi: repository/postgres).
type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, f ListFilter) ([]User, int, error)
	UpdateProfile(ctx context.Context, id, name, phone string) error
	UpdatePassword(ctx context.Context, id, hash string) error
	SetSuspended(ctx context.Context, id string, suspended bool) error
	CountByRole(ctx context.Context, role string) (int, error)
}

// PasswordHasher = port hashing (didefinisikan di kernel/security).
type PasswordHasher = security.PasswordHasher

// SessionRevoker = port untuk mencabut semua sesi user (implementasi di modul auth).
type SessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID string) error
}

// ---- aturan domain ----

func NormalizeEmail(email string) string { return normalize.Email(email) }

func ValidateName(f apperror.Fields, name string) string {
	name = strings.TrimSpace(name)
	switch n := len([]rune(name)); {
	case n < 2:
		f.Add("name", "Nama minimal 2 karakter")
	case n > 100:
		f.Add("name", "Nama maksimal 100 karakter")
	}
	return name
}

func ValidateEmail(f apperror.Fields, email string) string {
	email = NormalizeEmail(email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		f.Add("email", "Email tidak valid")
	}
	return email
}

func ValidatePassword(f apperror.Fields, field, pw string) {
	switch {
	case len(pw) < 8:
		f.Add(field, "Password minimal 8 karakter")
	case len(pw) > 72: // batas bcrypt
		f.Add(field, "Password maksimal 72 karakter")
	}
}

// NormalizePhone: "0812-3456 789" → "62812…" (lihat kernel/normalize).
func NormalizePhone(p string) string { return normalize.Phone(p) }
