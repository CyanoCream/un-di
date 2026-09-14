// Package stationtoken = implementasi domain.StationTokens (JWT HS256 terpisah dari token login).
//
// Kunci diturunkan dari JWT_SECRET dengan HMAC("checkin-station") dan issuer berbeda, sehingga token
// stasiun tidak pernah diterima middleware auth biasa, dan token login tidak bisa dipakai sebagai token stasiun.
package stationtoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"undangan/services/invitation/domain"
)

const issuer = "undangan-checkin"

var ErrInvalid = errors.New("station token invalid")

type Manager struct {
	key []byte
	ttl time.Duration
}

var _ domain.StationTokens = (*Manager)(nil)

func New(jwtSecret string, ttl time.Duration) *Manager {
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte("checkin-station"))
	return &Manager{key: mac.Sum(nil), ttl: ttl}
}

type claims struct {
	Station    string `json:"station"`
	PinVersion string `json:"pv"`
	gojwt.RegisteredClaims
}

func (m *Manager) Issue(invitationID, station, pinVersion string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.ttl)
	t := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims{
		Station: station, PinVersion: pinVersion,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer: issuer, Subject: invitationID,
			IssuedAt: gojwt.NewNumericDate(now), ExpiresAt: gojwt.NewNumericDate(exp),
		},
	})
	s, err := t.SignedString(m.key)
	return s, exp, err
}

func (m *Manager) Verify(token string) (string, string, string, error) {
	var c claims
	_, err := gojwt.ParseWithClaims(token, &c, func(*gojwt.Token) (any, error) { return m.key, nil },
		gojwt.WithValidMethods([]string{gojwt.SigningMethodHS256.Alg()}),
		gojwt.WithIssuer(issuer), gojwt.WithExpirationRequired())
	if err != nil || c.Subject == "" {
		return "", "", "", ErrInvalid
	}
	return c.Subject, c.Station, c.PinVersion, nil
}
