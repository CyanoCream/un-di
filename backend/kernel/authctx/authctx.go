// Package authctx membawa identitas user yang sudah terautentikasi di context request.
// Diisi oleh middleware modul auth; dibaca controller/service modul lain tanpa bergantung ke modul auth.
package authctx

import "context"

const (
	RoleSuperAdmin = "super_admin"
	RoleCustomer   = "customer"
)

type Principal struct {
	UserID string
	Role   string
	Name   string
	Email  string
}

func (p Principal) IsAdmin() bool { return p.Role == RoleSuperAdmin }

type key int

const (
	principalKey key = iota
	tokenErrKey
)

func With(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func From(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// MustFrom dipakai di handler yang sudah dijaga router (Auth/Role).
func MustFrom(ctx context.Context) Principal {
	p, _ := From(ctx)
	return p
}

// WithTokenError menyimpan alasan token ditolak (mis. "token_expired") supaya
// router bisa membalas 401 dengan kode yang tepat → frontend melakukan refresh.
func WithTokenError(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, tokenErrKey, code)
}

func TokenError(ctx context.Context) string {
	s, _ := ctx.Value(tokenErrKey).(string)
	return s
}
