package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/billing/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

type Subscriptions struct{ pool *pgxpool.Pool }

var _ domain.SubscriptionRepository = (*Subscriptions)(nil)

func NewSubscriptions(pool *pgxpool.Pool) *Subscriptions { return &Subscriptions{pool: pool} }

const subSelect = `SELECT s.id, s.user_id, u.name, u.email, s.plan_id, p.name, s.status::text, s.starts_at, s.ends_at,
		s.grace_ends_at, s.max_invitations, s.max_guests, s.allow_checkin, s.created_at
	FROM subscriptions s
	JOIN users u ON u.id = s.user_id
	JOIN plans p ON p.id = s.plan_id`

func scanSub(row pgx.Row) (*domain.Subscription, error) {
	var s domain.Subscription
	if err := row.Scan(&s.ID, &s.UserID, &s.UserName, &s.UserEmail, &s.PlanID, &s.PlanName, &s.Status, &s.StartsAt,
		&s.EndsAt, &s.GraceEndsAt, &s.MaxInvitations, &s.MaxGuests, &s.AllowCheckin, &s.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &s, nil
}

func (r *Subscriptions) query(ctx context.Context, sql string, args ...any) ([]domain.Subscription, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, notFound(err)
	}
	defer rows.Close()
	var out []domain.Subscription
	for rows.Next() {
		s, err := scanSub(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (r *Subscriptions) Create(ctx context.Context, s *domain.Subscription) error {
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, status, starts_at, ends_at, grace_ends_at, max_invitations, max_guests, allow_checkin)
		VALUES ($1, $2, $3::subscription_status, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`,
		s.UserID, s.PlanID, s.Status, s.StartsAt, s.EndsAt, s.GraceEndsAt, s.MaxInvitations, s.MaxGuests, s.AllowCheckin).Scan(&s.ID, &s.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "23503" || pgErr.Code == "22P02") {
		if pgErr.ConstraintName == "subscriptions_plan_id_fkey" {
			return apperror.NotFound("Paket tidak ditemukan")
		}
		return apperror.NotFound("User tidak ditemukan")
	}
	return err
}

func (r *Subscriptions) Update(ctx context.Context, s *domain.Subscription) error {
	return execOne(ctx, r.pool, `
		UPDATE subscriptions SET plan_id = $2, status = $3::subscription_status, starts_at = $4, ends_at = $5,
			grace_ends_at = $6, max_invitations = $7, max_guests = $8, allow_checkin = $9, updated_at = now()
		WHERE id = $1`,
		s.ID, s.PlanID, s.Status, s.StartsAt, s.EndsAt, s.GraceEndsAt, s.MaxInvitations, s.MaxGuests, s.AllowCheckin)
}

func (r *Subscriptions) FindByID(ctx context.Context, id string) (*domain.Subscription, error) {
	return scanSub(database.Conn(ctx, r.pool).QueryRow(ctx, subSelect+` WHERE s.id = $1`, id))
}

func (r *Subscriptions) FindCurrentForUser(ctx context.Context, userID string) (*domain.Subscription, error) {
	return scanSub(database.Conn(ctx, r.pool).QueryRow(ctx, subSelect+`
		WHERE s.user_id = $1 AND s.status IN ('active', 'grace')
		ORDER BY (s.status = 'active') DESC, s.ends_at DESC LIMIT 1`, userID))
}

func (r *Subscriptions) ListByUser(ctx context.Context, userID string) ([]domain.Subscription, error) {
	return r.query(ctx, subSelect+` WHERE s.user_id = $1 ORDER BY s.created_at DESC`, userID)
}

func (r *Subscriptions) List(ctx context.Context, f domain.SubscriptionFilter) ([]domain.Subscription, int, error) {
	const where = ` WHERE ($1 = '' OR s.status::text = $1)
		AND ($2 = '' OR u.name ILIKE '%' || $2 || '%' OR u.email ILIKE '%' || $2 || '%')`
	var total int
	if err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM subscriptions s
		JOIN users u ON u.id = s.user_id`+where, f.Status, f.Query).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := r.query(ctx, subSelect+where+` ORDER BY s.created_at DESC LIMIT $3 OFFSET $4`,
		f.Status, f.Query, f.Limit, f.Offset)
	return items, total, err
}

func (r *Subscriptions) HasLive(ctx context.Context, userID string) (bool, error) {
	var ok bool
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM subscriptions
		WHERE user_id = $1 AND status IN ('active', 'grace'))`, userID).Scan(&ok)
	return ok, notFound(err)
}

func (r *Subscriptions) ListByStatusEndingBefore(ctx context.Context, status string, t time.Time) ([]domain.Subscription, error) {
	switch status {
	case domain.SubscriptionActive:
		return r.query(ctx, subSelect+` WHERE s.status = 'active' AND s.ends_at < $1 ORDER BY s.ends_at`, t)
	case domain.SubscriptionGrace:
		return r.query(ctx, subSelect+` WHERE s.status = 'grace' AND s.grace_ends_at < $1 ORDER BY s.grace_ends_at`, t)
	default:
		return nil, fmt.Errorf("ListByStatusEndingBefore: status %q tidak didukung", status)
	}
}

func (r *Subscriptions) TransitionStatus(ctx context.Context, id, from, to string) (bool, error) {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE subscriptions SET status = $3::subscription_status, updated_at = now()
		WHERE id = $1 AND status::text = $2`, id, from, to)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
