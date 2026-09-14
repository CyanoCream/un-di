// Package domain (media) = entity file upload & pustaka musik, aturan validasi, kontrak repository.
package domain

import (
	"context"
	"strings"
	"time"

	"undangan/kernel/apperror"
)

const (
	KindImage = "image"
	KindAudio = "audio"
)

// Upload = jejak file publik yang diunggah user (tabel uploads).
type Upload struct {
	ID        string    `json:"id"`
	UserID    *string   `json:"user_id"`
	Kind      string    `json:"kind"` // image | audio
	Path      string    `json:"path"` // key di FileStorage
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// MusicTrack = lagu di pustaka musik yang bisa dipilih untuk undangan.
type MusicTrack struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Artist          string    `json:"artist"`
	URL             string    `json:"url"`
	DurationSeconds int       `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}

type UploadRepository interface {
	Create(ctx context.Context, u *Upload) error
}

type MusicRepository interface {
	Create(ctx context.Context, t *MusicTrack) error
	List(ctx context.Context) ([]MusicTrack, error)
	// SoftDelete mengembalikan apperror.ErrNotFound bila lagu tidak ada / sudah dihapus.
	SoftDelete(ctx context.Context, id string) error
}

// ---- aturan domain ----

func ValidateKind(f apperror.Fields, kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	switch kind {
	case "":
		return KindImage
	case KindImage, KindAudio:
		return kind
	default:
		f.Add("kind", "Jenis file harus image atau audio")
		return kind
	}
}

func ValidateTitle(f apperror.Fields, title string) string {
	title = strings.TrimSpace(title)
	switch n := len([]rune(title)); {
	case n == 0:
		f.Add("title", "Judul wajib diisi")
	case n > 100:
		f.Add("title", "Judul maksimal 100 karakter")
	}
	return title
}

func ValidateArtist(f apperror.Fields, artist string) string {
	artist = strings.TrimSpace(artist)
	if len([]rune(artist)) > 100 {
		f.Add("artist", "Nama artis maksimal 100 karakter")
	}
	return artist
}
