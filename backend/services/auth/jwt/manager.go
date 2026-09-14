// Package jwt = implementasi domain.TokenManager dengan HS256.
package jwt

import (
	"errors"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"undangan/services/auth/domain"
	"undangan/kernel/security"
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

const issuer = "undangan-api"

type claims struct {
	Role   string `json:"role"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Portal string `json:"portal"`
	gojwt.RegisteredClaims
}

type HS256Manager struct {
	secret []byte
	ttl    time.Duration
}

var _ domain.TokenManager = (*HS256Manager)(nil)

func NewHS256(secret string, ttl time.Duration) (*HS256Manager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET minimal 32 karakter")
	}
	return &HS256Manager{secret: []byte(secret), ttl: ttl}, nil
}

func (m *HS256Manager) Issue(c domain.Claims) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.ttl)
	t := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims{
		Role: c.Role, Name: c.Name, Email: c.Email, Portal: c.Portal,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   c.UserID,
			Audience:  gojwt.ClaimStrings{c.Portal},
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now.Add(-30 * time.Second)),
			ExpiresAt: gojwt.NewNumericDate(exp),
			ID:        security.RandomToken(12),
		},
	})
	s, err := t.SignedString(m.secret)
	return s, exp, err
}

func (m *HS256Manager) Verify(token string) (domain.Claims, error) {
	var c claims
	_, err := gojwt.ParseWithClaims(token, &c, func(t *gojwt.Token) (any, error) {
		return m.secret, nil
	},
		gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}), // tolak alg "none"/RS
		gojwt.WithIssuer(issuer),
		gojwt.WithExpirationRequired(),
		gojwt.WithLeeway(10*time.Second),
	)
	switch {
	case errors.Is(err, gojwt.ErrTokenExpired):
		return domain.Claims{}, ErrTokenExpired
	case err != nil:
		return domain.Claims{}, ErrTokenInvalid
	}
	return domain.Claims{UserID: c.Subject, Role: c.Role, Name: c.Name, Email: c.Email, Portal: c.Portal}, nil
}
