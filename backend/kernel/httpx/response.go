// Package httpx = utilitas HTTP bersama: respons JSON, pemetaan error, decode, paginasi, router, middleware.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"undangan/kernel/apperror"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, v any)      { JSON(w, http.StatusOK, v) }
func Created(w http.ResponseWriter, v any) { JSON(w, http.StatusCreated, v) }
func NoContent(w http.ResponseWriter)      { w.WriteHeader(http.StatusNoContent) }

// Fail menerjemahkan error ke respons. Error non-apperror dicatat dan disembunyikan dari user.
func Fail(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apperror.Error
	if !errors.As(err, &ae) {
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		ae = &apperror.Error{Status: http.StatusInternalServerError, Code: "internal", Message: "Terjadi kesalahan pada server"}
	}
	body := map[string]any{"code": ae.Code, "message": ae.Message}
	if len(ae.Fields) > 0 {
		body["fields"] = ae.Fields
	}
	JSON(w, ae.Status, map[string]any{"error": body})
}

// Page = parameter paginasi dari query ?page=&per_page=.
type Page struct {
	Page, PerPage int
}

func (p Page) Offset() int { return (p.Page - 1) * p.PerPage }

type Paginated[T any] struct {
	Items   []T `json:"items"`
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func NewPaginated[T any](items []T, total int, p Page) Paginated[T] {
	if items == nil {
		items = []T{}
	}
	return Paginated[T]{Items: items, Total: total, Page: p.Page, PerPage: p.PerPage}
}

type List[T any] struct {
	Items []T `json:"items"`
}

func NewList[T any](items []T) List[T] {
	if items == nil {
		items = []T{}
	}
	return List[T]{Items: items}
}
