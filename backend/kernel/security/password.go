// Package security berisi implementasi kriptografi yang dipakai beberapa modul.
package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher = port hashing password (dipakai service user, auth, invitation).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

var _ PasswordHasher = (*BcryptHasher)(nil)

// BcryptHasher mengimplementasikan PasswordHasher.
type BcryptHasher struct{ Cost int }

func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{Cost: 12} }

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.Cost)
	return string(b), err
}

func (h *BcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// RandomToken = token acak URL-safe dengan n byte entropi.
func RandomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func SHA256(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

// RandomCode: kode huruf besar/angka tanpa karakter ambigu (0/O, 1/I/L).
func RandomCode(n int) string {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}
