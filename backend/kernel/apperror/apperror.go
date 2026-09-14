// Package apperror mendefinisikan error domain yang aman ditampilkan ke user.
// Service mengembalikan *Error; controller menerjemahkannya ke HTTP lewat httpx.
package apperror

import (
	"errors"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Is: dua *Error dianggap sama jika Code sama (errors.Is(err, apperror.ErrNotFound)).
func (e *Error) Is(target error) bool {
	var t *Error
	return errors.As(target, &t) && t.Code == e.Code
}

// ErrNotFound dikembalikan repository ketika baris tidak ada.
var ErrNotFound = &Error{Status: http.StatusNotFound, Code: "not_found", Message: "Data tidak ditemukan"}

func NotFound(msg string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: msg}
}

func BadRequest(msg string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: "bad_request", Message: msg}
}

func Unauthorized(code, msg string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: code, Message: msg}
}

func Forbidden(msg string) *Error {
	return &Error{Status: http.StatusForbidden, Code: "forbidden", Message: msg}
}

func Conflict(code, msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: msg}
}

func Unprocessable(code, msg string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: code, Message: msg}
}

func TooManyRequests() *Error {
	return &Error{Status: http.StatusTooManyRequests, Code: "rate_limited", Message: "Terlalu banyak percobaan, coba lagi beberapa saat lagi"}
}

// Validation: fields = nama field → pesan. Message = pesan pertama yang ditemukan.
func Validation(fields map[string]string) *Error {
	msg := "Data belum valid"
	for _, v := range fields {
		msg = v
		break
	}
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation", Message: msg, Fields: fields}
}

// Fields membantu mengumpulkan error validasi.
type Fields map[string]string

func (f Fields) Add(field, msg string) {
	if _, exists := f[field]; !exists {
		f[field] = msg
	}
}

func (f Fields) Err() error {
	if len(f) == 0 {
		return nil
	}
	return Validation(f)
}
