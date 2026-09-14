// Package repository (media) = implementasi Postgres untuk UploadRepository & MusicRepository.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/media/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

type Uploads struct{ pool *pgxpool.Pool }

var _ domain.UploadRepository = (*Uploads)(nil)

func NewUploads(pool *pgxpool.Pool) *Uploads { return &Uploads{pool: pool} }

func (r *Uploads) Create(ctx context.Context, u *domain.Upload) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO uploads (user_id, kind, path, size_bytes) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`, u.UserID, u.Kind, u.Path, u.SizeBytes).Scan(&u.ID, &u.CreatedAt)
}

type Music struct{ pool *pgxpool.Pool }

var _ domain.MusicRepository = (*Music)(nil)

func NewMusic(pool *pgxpool.Pool) *Music { return &Music{pool: pool} }

func (r *Music) Create(ctx context.Context, t *domain.MusicTrack) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO music_tracks (title, artist, url, duration_seconds) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`, t.Title, t.Artist, t.URL, t.DurationSeconds).Scan(&t.ID, &t.CreatedAt)
}

func (r *Music) List(ctx context.Context) ([]domain.MusicTrack, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `
		SELECT id, title, artist, url, duration_seconds, created_at
		FROM music_tracks WHERE deleted_at IS NULL ORDER BY created_at DESC, title`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MusicTrack
	for rows.Next() {
		var t domain.MusicTrack
		if err := rows.Scan(&t.ID, &t.Title, &t.Artist, &t.URL, &t.DurationSeconds, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Music) SoftDelete(ctx context.Context, id string) error {
	// id bukan UUID → anggap tidak ada (tanpa query, supaya transaksi aktif tidak ikut batal).
	var uid pgtype.UUID
	if err := uid.Scan(id); err != nil || !uid.Valid {
		return apperror.ErrNotFound
	}
	tag, err := database.Conn(ctx, r.pool).Exec(ctx,
		`UPDATE music_tracks SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, uid)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
