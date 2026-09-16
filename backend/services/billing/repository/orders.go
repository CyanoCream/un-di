package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/database"
	"undangan/services/billing/domain"
)

type Orders struct{ pool *pgxpool.Pool }

var _ domain.OrderRepository = (*Orders)(nil)

func NewOrders(pool *pgxpool.Pool) *Orders { return &Orders{pool: pool} }

const orderSelect = `SELECT o.id, o.code, o.user_id, u.name, u.email, o.plan_id, p.name, o.price, o.unique_code, o.amount,
		o.status::text, COALESCE(o.proof_path, ''), o.proof_uploaded_at, o.reject_reason, o.reviewed_by, rv.name,
		o.reviewed_at, o.subscription_id, o.expires_at, o.created_at
	FROM orders o
	JOIN users u ON u.id = o.user_id
	JOIN plans p ON p.id = o.plan_id
	LEFT JOIN users rv ON rv.id = o.reviewed_by`

func scanOrder(row pgx.Row) (*domain.Order, error) {
	var o domain.Order
	if err := row.Scan(&o.ID, &o.Code, &o.UserID, &o.UserName, &o.UserEmail, &o.PlanID, &o.PlanName, &o.Price,
		&o.UniqueCode, &o.Amount, &o.Status, &o.ProofPath, &o.ProofUploadedAt, &o.RejectReason, &o.ReviewedBy,
		&o.ReviewedByName, &o.ReviewedAt, &o.SubscriptionID, &o.ExpiresAt, &o.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &o, nil
}

func (r *Orders) query(ctx context.Context, sql string, args ...any) ([]domain.Order, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, sql, args...)
	if err != nil {
		return nil, notFound(err)
	}
	defer rows.Close()
	var out []domain.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *Orders) Create(ctx context.Context, o *domain.Order) error {
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO orders (code, user_id, plan_id, price, unique_code, amount, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::order_status, $8)
		ON CONFLICT (code) DO NOTHING
		RETURNING id, created_at`,
		o.Code, o.UserID, o.PlanID, o.Price, o.UniqueCode, o.Amount, o.Status, o.ExpiresAt).Scan(&o.ID, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrOrderCodeTaken
	}
	return err
}

func (r *Orders) Update(ctx context.Context, o *domain.Order) error {
	return execOne(ctx, r.pool, `
		UPDATE orders SET status = $2::order_status, proof_path = NULLIF($3, ''), proof_uploaded_at = $4,
			reject_reason = $5, reviewed_by = $6, reviewed_at = $7, subscription_id = $8, updated_at = now()
		WHERE id = $1`,
		o.ID, o.Status, o.ProofPath, o.ProofUploadedAt, o.RejectReason, o.ReviewedBy, o.ReviewedAt, o.SubscriptionID)
}

func (r *Orders) FindByID(ctx context.Context, id string) (*domain.Order, error) {
	return scanOrder(database.Conn(ctx, r.pool).QueryRow(ctx, orderSelect+` WHERE o.id = $1`, id))
}

func (r *Orders) FindByIDForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	return scanOrder(database.Conn(ctx, r.pool).QueryRow(ctx, orderSelect+` WHERE o.id = $1 FOR UPDATE OF o`, id))
}

func (r *Orders) List(ctx context.Context, f domain.OrderFilter) ([]domain.Order, int, error) {
	const where = ` WHERE ($1 = '' OR o.user_id::text = $1)
		AND ($2 = '' OR o.status::text = $2)
		AND ($3 = '' OR u.name ILIKE '%' || $3 || '%' OR u.email ILIKE '%' || $3 || '%' OR o.code ILIKE '%' || $3 || '%')`
	var total int
	if err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM orders o JOIN users u ON u.id = o.user_id`+where,
		f.UserID, f.Status, f.Query).Scan(&total); err != nil {
		return nil, 0, err
	}
	items, err := r.query(ctx, orderSelect+where+` ORDER BY o.created_at DESC LIMIT $4 OFFSET $5`,
		f.UserID, f.Status, f.Query, f.Limit, f.Offset)
	return items, total, err
}

func (r *Orders) ListPendingByUser(ctx context.Context, userID string) ([]domain.Order, error) {
	return r.query(ctx, orderSelect+` WHERE o.user_id = $1 AND o.status IN ('awaiting_payment', 'awaiting_confirmation')
		ORDER BY o.created_at DESC`, userID)
}

func (r *Orders) TakenAmounts(ctx context.Context, lo, hi int64, now time.Time) ([]int64, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `SELECT amount FROM orders
		WHERE amount BETWEEN $1 AND $2
		  AND (status = 'awaiting_confirmation' OR (status = 'awaiting_payment' AND expires_at > $3))`, lo, hi, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[int64])
}

func (r *Orders) ExpireOverdue(ctx context.Context, now time.Time) (int, error) {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE orders SET status = 'expired', updated_at = now()
		WHERE status = 'awaiting_payment' AND expires_at < $1`, now)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
