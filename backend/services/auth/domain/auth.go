// Package domain (auth) = autentikasi berbasis JWT: access token (JWT, pendek) + refresh token (opaque, dirotasi).
package domain

import (
	"context"
	"time"

	"undangan/kernel/authctx"
)

const (
	PortalAdmin    = "admin"
	PortalCustomer = "customer"
)

// Claims = isi access token JWT.
type Claims struct {
	UserID string
	Role   string
	Name   string
	Email  string
	Portal string
}

func (c Claims) Principal() authctx.Principal {
	return authctx.Principal{UserID: c.UserID, Role: c.Role, Name: c.Name, Email: c.Email}
}

type TokenPair struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// TokenManager = port penerbit & pemverifikasi JWT (implementasi: jwt.HS256Manager).
type TokenManager interface {
	Issue(c Claims) (token string, expiresAt time.Time, err error)
	// Verify mengembalikan ErrTokenExpired bila kedaluwarsa, ErrTokenInvalid bila rusak/palsu.
	Verify(token string) (Claims, error)
}

type RefreshToken struct {
	ID        string
	UserID    string
	FamilyID  string
	TokenHash []byte
	Portal    string
	IP        string
	UserAgent string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, t *RefreshToken) error
	// FindByHashForUpdate mengunci baris supaya rotasi paralel tidak lolos dua kali.
	FindByHashForUpdate(ctx context.Context, hash []byte) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeFamily(ctx context.Context, familyID string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// Account = data akun yang dibutuhkan autentikasi (bentuk JSON sama dengan user).
type Account struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	IsSuspended  bool      `json:"is_suspended"`
	CreatedAt    time.Time `json:"created_at"`
	PasswordHash string    `json:"-"`
}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// AccountStore = port ke service user. Saat monolith diisi adapter in-process; saat microservice diganti client API.
type AccountStore interface {
	FindByEmail(ctx context.Context, email string) (*Account, error) // apperror.ErrNotFound bila tidak ada
	FindByID(ctx context.Context, id string) (*Account, error)
	Register(ctx context.Context, in RegisterInput) (*Account, error)
}

// ClientInfo = metadata perangkat untuk refresh token.
type ClientInfo struct {
	IP        string
	UserAgent string
}
