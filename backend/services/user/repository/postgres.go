// Package repository (user) = implementasi Postgres untuk domain.Repository.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/services/user/domain"
)

type Postgres struct{ pool *pgxpool.Pool }

var _ domain.Repository = (*Postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

const cols = `id, name, email, phone, role, is_suspended, created_at, password_hash`

func scan(row pgx.Row) (*domain.User, error) {
	var u domain.User
	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.Role, &u.IsSuspended, &u.CreatedAt, &u.PasswordHash); err != nil {
		return nil, database.NotFound(err)
	}
	return &u, nil
}

func (r *Postgres) Create(ctx context.Context, u *domain.User) error {
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO users (name, email, phone, password_hash, role) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role).Scan(&u.ID, &u.CreatedAt)
	if database.IsUniqueViolation(err) {
		return apperror.Validation(map[string]string{"email": "Email sudah terdaftar"})
	}
	return err
}

func (r *Postgres) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return scan(database.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT `+cols+` FROM users WHERE id = $1 AND deleted_at IS NULL`, id))
}

func (r *Postgres) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scan(database.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT `+cols+` FROM users WHERE lower(email) = lower($1) AND deleted_at IS NULL`, email))
}

func (r *Postgres) List(ctx context.Context, f domain.ListFilter) ([]domain.User, int, error) {
	const where = `deleted_at IS NULL
		AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR email ILIKE '%' || $1 || '%' OR phone LIKE '%' || $1 || '%')
		AND ($2 = '' OR role::text = $2)`
	conn := database.Conn(ctx, r.pool)

	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM users WHERE `+where, f.Query, f.Role).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.Query(ctx, `SELECT `+cols+` FROM users WHERE `+where+` ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		f.Query, f.Role, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *u)
	}
	return out, total, rows.Err()
}

func (r *Postgres) exec(ctx context.Context, sql string, args ...any) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *Postgres) UpdateProfile(ctx context.Context, id, name, phone string) error {
	return r.exec(ctx, `UPDATE users SET name = $2, phone = $3, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, name, phone)
}

func (r *Postgres) UpdatePassword(ctx context.Context, id, hash string) error {
	return r.exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, hash)
}

func (r *Postgres) SetSuspended(ctx context.Context, id string, suspended bool) error {
	return r.exec(ctx, `UPDATE users SET is_suspended = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, suspended)
}

func (r *Postgres) CountByRole(ctx context.Context, role string) (int, error) {
	var n int
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM users WHERE role::text = $1 AND deleted_at IS NULL`, role).Scan(&n)
	return n, err
}
