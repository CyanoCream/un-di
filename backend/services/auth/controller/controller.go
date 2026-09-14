// Package controller (auth) = endpoint login/register/refresh/logout + middleware JWT.
package controller

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"undangan/services/auth/domain"
	"undangan/services/auth/jwt"
	"undangan/services/auth/service"
	"undangan/kernel/normalize"
	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
	"undangan/kernel/authctx"
)

const (
	AccessCookie  = "undangan_at"
	RefreshCookie = "undangan_rt"
	refreshPath   = "/api/v1/auth"
)

type Controller struct {
	svc          service.Service
	accounts     domain.AccountStore
	secureCookie bool
	loginLimit   *httpx.RateLimiter
	registerLim  *httpx.RateLimiter
}

func New(svc service.Service, accounts domain.AccountStore, secureCookie bool) *Controller {
	return &Controller{
		svc: svc, accounts: accounts, secureCookie: secureCookie,
		loginLimit:  httpx.NewRateLimiter(10, 5*time.Minute),
		registerLim: httpx.NewRateLimiter(5, time.Hour),
	}
}

func (c *Controller) Register(rt *httpx.Router) {
	rt.Public("POST /api/v1/auth/login", c.login)
	rt.Public("POST /api/v1/auth/register", c.register)
	rt.Public("POST /api/v1/auth/refresh", c.refresh)
	rt.Public("POST /api/v1/auth/logout", c.logout)
	rt.Auth("GET /api/v1/auth/me", c.me)
}

// Middleware membaca access token (header Authorization: Bearer atau cookie) → authctx.
// Tidak menolak request; penolakan dilakukan Router sesuai kebutuhan endpoint.
func (c *Controller) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		} else if ck, err := r.Cookie(AccessCookie); err == nil {
			token = ck.Value
		}
		if token != "" {
			claims, err := c.svc.Authenticate(token)
			switch {
			case err == nil:
				r = r.WithContext(authctx.With(r.Context(), claims.Principal()))
			case errors.Is(err, jwt.ErrTokenExpired):
				r = r.WithContext(authctx.WithTokenError(r.Context(), "token_expired"))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func client(r *http.Request) domain.ClientInfo {
	return domain.ClientInfo{IP: httpx.ClientIP(r), UserAgent: r.UserAgent()}
}

func (c *Controller) setCookies(w http.ResponseWriter, p domain.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name: AccessCookie, Value: p.AccessToken, Path: "/", Expires: p.AccessExpiresAt,
		HttpOnly: true, Secure: c.secureCookie, SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: RefreshCookie, Value: p.RefreshToken, Path: refreshPath, Expires: p.RefreshExpiresAt,
		HttpOnly: true, Secure: c.secureCookie, SameSite: http.SameSiteStrictMode,
	})
}

func (c *Controller) clearCookies(w http.ResponseWriter) {
	for _, ck := range []*http.Cookie{
		{Name: AccessCookie, Path: "/"},
		{Name: RefreshCookie, Path: refreshPath},
	} {
		ck.MaxAge, ck.HttpOnly, ck.Secure = -1, true, c.secureCookie
		http.SetCookie(w, ck)
	}
}

func authResponse(w http.ResponseWriter, status int, u *domain.Account, p domain.TokenPair) {
	// Token juga dikirim di body untuk klien non-browser (mobile); SPA cukup memakai cookie HttpOnly.
	httpx.JSON(w, status, map[string]any{
		"user":               u,
		"access_token":       p.AccessToken,
		"access_expires_at":  p.AccessExpiresAt,
		"refresh_expires_at": p.RefreshExpiresAt,
	})
}

func (c *Controller) login(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Email, Password, Portal string }
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if !c.loginLimit.Allow("ip:"+httpx.ClientIP(r)) || !c.loginLimit.Allow("email:"+normalize.Email(in.Email)) {
		return apperror.TooManyRequests()
	}
	u, pair, err := c.svc.Login(r.Context(), in.Email, in.Password, in.Portal, client(r))
	if err != nil {
		return err
	}
	c.setCookies(w, pair)
	authResponse(w, http.StatusOK, u, pair)
	return nil
}

func (c *Controller) register(w http.ResponseWriter, r *http.Request) error {
	var in domain.RegisterInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	if !c.registerLim.Allow(httpx.ClientIP(r)) {
		return apperror.TooManyRequests()
	}
	u, pair, err := c.svc.Register(r.Context(), in, client(r))
	if err != nil {
		return err
	}
	c.setCookies(w, pair)
	authResponse(w, http.StatusCreated, u, pair)
	return nil
}

func (c *Controller) refresh(w http.ResponseWriter, r *http.Request) error {
	raw := ""
	if ck, err := r.Cookie(RefreshCookie); err == nil {
		raw = ck.Value
	} else {
		var in struct {
			RefreshToken string `json:"refresh_token"`
		}
		_ = httpx.Decode(r, &in)
		raw = in.RefreshToken
	}
	u, pair, err := c.svc.Refresh(r.Context(), raw, client(r))
	if err != nil {
		c.clearCookies(w)
		return err
	}
	c.setCookies(w, pair)
	authResponse(w, http.StatusOK, u, pair)
	return nil
}

func (c *Controller) logout(w http.ResponseWriter, r *http.Request) error {
	if ck, err := r.Cookie(RefreshCookie); err == nil {
		if err := c.svc.Logout(r.Context(), ck.Value); err != nil {
			return err
		}
	}
	c.clearCookies(w)
	httpx.NoContent(w)
	return nil
}

func (c *Controller) me(w http.ResponseWriter, r *http.Request) error {
	// Ambil dari DB (bukan dari klaim JWT) supaya perubahan profil langsung terlihat.
	u, err := c.accounts.FindByID(r.Context(), authctx.MustFrom(r.Context()).UserID)
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.Unauthorized("unauthorized", "Akun tidak ditemukan")
	}
	if err != nil {
		return err
	}
	httpx.OK(w, map[string]any{"user": u})
	return nil
}
