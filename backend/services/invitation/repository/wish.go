package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/invitation/domain"
	"undangan/kernel/database"
)

type Wishes struct{ pool *pgxpool.Pool }

var _ domain.WishRepository = (*Wishes)(nil)

func NewWishes(pool *pgxpool.Pool) *Wishes { return &Wishes{pool: pool} }

const wishCols = `id, invitation_id, guest_id, name, attendance, pax, message, is_hidden, created_at`

func scanWish(row pgx.Row) (*domain.Wish, error) {
	var w domain.Wish
	if err := row.Scan(&w.ID, &w.InvitationID, &w.GuestID, &w.Name, &w.Attendance, &w.Pax, &w.Message, &w.IsHidden, &w.CreatedAt); err != nil {
		return nil, database.NotFound(err)
	}
	return &w, nil
}

func (r *Wishes) Create(ctx context.Context, w *domain.Wish) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO wishes (invitation_id, guest_id, name, attendance, pax, message, ip) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`, w.InvitationID, w.GuestID, w.Name, w.Attendance, w.Pax, w.Message, w.IP).Scan(&w.ID, &w.CreatedAt)
}

func (r *Wishes) List(ctx context.Context, invID string, includeHidden bool, limit, offset int) ([]domain.Wish, int, error) {
	const where = ` WHERE invitation_id = $1 AND deleted_at IS NULL AND ($2 OR NOT is_hidden)`
	conn := database.Conn(ctx, r.pool)
	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM wishes`+where, invID, includeHidden).Scan(&total); err != nil {
		return nil, 0, database.NotFound(err)
	}
	rows, err := conn.Query(ctx, `SELECT `+wishCols+` FROM wishes`+where+` ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		invID, includeHidden, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Wish
	for rows.Next() {
		w, err := scanWish(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *w)
	}
	return out, total, rows.Err()
}

func (r *Wishes) Stats(ctx context.Context, invID string, includeHidden bool) (domain.WishStats, error) {
	var s domain.WishStats
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE attendance = 'hadir'), count(*) FILTER (WHERE attendance = 'tidak'),
			count(*) FILTER (WHERE attendance = 'ragu'), COALESCE(sum(pax) FILTER (WHERE attendance = 'hadir'), 0)
		FROM wishes WHERE invitation_id = $1 AND deleted_at IS NULL AND ($2 OR NOT is_hidden)`, invID, includeHidden).
		Scan(&s.Hadir, &s.Tidak, &s.Ragu, &s.TotalPax)
	return s, database.NotFound(err)
}

func (r *Wishes) SetHidden(ctx context.Context, invID, id string, hidden bool) (*domain.Wish, error) {
	return scanWish(database.Conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE wishes SET is_hidden = $3 WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL
		RETURNING `+wishCols, id, invID, hidden))
}

func (r *Wishes) SoftDelete(ctx context.Context, invID, id string) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx,
		`UPDATE wishes SET deleted_at = now() WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, id, invID)
	if err != nil {
		return database.NotFound(err)
	}
	return affected(nil, tag.RowsAffected())
}
