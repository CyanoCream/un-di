package httpx

import (
	"net/http"

	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
)

// HandlerFunc = handler controller yang boleh mengembalikan error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

func Handle(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			Fail(w, r, err)
		}
	}
}

// Router membungkus ServeMux dengan penjaga akses.
// Autentikasi (JWT → authctx) dipasang sebagai middleware global; router hanya memeriksa hasilnya.
type Router struct {
	mux *http.ServeMux
}

func NewRouter(mux *http.ServeMux) *Router { return &Router{mux: mux} }

// Public: tanpa login.
func (rt *Router) Public(pattern string, h HandlerFunc) {
	rt.mux.HandleFunc(pattern, Handle(h))
}

// Auth: login dengan role apa pun.
func (rt *Router) Auth(pattern string, h HandlerFunc) {
	rt.mux.HandleFunc(pattern, Handle(func(w http.ResponseWriter, r *http.Request) error {
		if _, err := requirePrincipal(r); err != nil {
			return err
		}
		return h(w, r)
	}))
}

// Role: login dengan salah satu role yang disebut.
func (rt *Router) Role(pattern string, h HandlerFunc, roles ...string) {
	rt.mux.HandleFunc(pattern, Handle(func(w http.ResponseWriter, r *http.Request) error {
		p, err := requirePrincipal(r)
		if err != nil {
			return err
		}
		for _, role := range roles {
			if p.Role == role {
				return h(w, r)
			}
		}
		return apperror.Forbidden("Anda tidak memiliki akses ke fitur ini")
	}))
}

func (rt *Router) Admin(pattern string, h HandlerFunc) {
	rt.Role(pattern, h, authctx.RoleSuperAdmin)
}

func (rt *Router) Customer(pattern string, h HandlerFunc) {
	rt.Role(pattern, h, authctx.RoleCustomer)
}

func requirePrincipal(r *http.Request) (authctx.Principal, error) {
	p, ok := authctx.From(r.Context())
	if ok {
		return p, nil
	}
	if code := authctx.TokenError(r.Context()); code == "token_expired" {
		return p, apperror.Unauthorized(code, "Sesi kedaluwarsa")
	}
	return p, apperror.Unauthorized("unauthorized", "Silakan masuk terlebih dahulu")
}
