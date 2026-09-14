package controller

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"undangan/services/media/domain"
	"undangan/kernel/authctx"
)

type memStorage struct{ fsys fs.FS }

func (m memStorage) SavePublic(context.Context, []byte, string) (string, string, error) {
	return "", "", nil
}
func (m memStorage) SavePrivate(context.Context, string, []byte, string) (string, error) {
	return "", nil
}
func (m memStorage) OpenPrivate(context.Context, string) (io.ReadSeekCloser, time.Time, error) {
	return nil, time.Time{}, fs.ErrNotExist
}
func (m memStorage) PublicFS() fs.FS { return m.fsys }

type nopService struct{}

func (nopService) Upload(context.Context, authctx.Principal, string, []byte) (string, error) {
	return "", nil
}
func (nopService) ListMusic(context.Context) ([]domain.MusicTrack, error) { return nil, nil }
func (nopService) CreateMusic(context.Context, authctx.Principal, string, string, []byte) (*domain.MusicTrack, error) {
	return nil, nil
}
func (nopService) DeleteMusic(context.Context, authctx.Principal, string) error { return nil }

func TestServeUploads(t *testing.T) {
	c := New(nopService{}, memStorage{fstest.MapFS{"2026/09/a.png": {Data: []byte("\x89PNG\r\n\x1a\n")}}})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /uploads/{path...}", c.ServeUploads)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/uploads/2026/09/a.png", nil))
	if rec.Code != 200 || rec.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" ||
		rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("unexpected response %d %v", rec.Code, rec.Header())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/uploads/2026/09/missing.png", nil))
	if rec.Code != 404 || rec.Header().Get("Cache-Control") == "public, max-age=31536000, immutable" {
		t.Fatalf("missing file: %d %v", rec.Code, rec.Header())
	}
}
