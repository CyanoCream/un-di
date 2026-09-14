// Package repository (dashboard) = query statistik admin di Postgres.
package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/dashboard/domain"
	"undangan/kernel/database"
)

type Postgres struct{ pool *pgxpool.Pool }

var _ domain.Repository = (*Postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

const statsSQL = `
SELECT
	(SELECT count(*) FROM users WHERE role = 'customer' AND deleted_at IS NULL),
	(SELECT count(*) FROM subscriptions WHERE status = 'active'),
	(SELECT count(*) FROM subscriptions WHERE status = 'grace'),
	(SELECT count(*) FROM orders WHERE status = 'awaiting_confirmation'),
	(SELECT count(*) FROM invitations WHERE deleted_at IS NULL),
	(SELECT count(*) FROM invitations WHERE status = 'published' AND deleted_at IS NULL),
	(SELECT coalesce(sum(amount), 0)::bigint FROM orders
		WHERE status = 'paid' AND reviewed_at >= date_trunc('month', now())),
	(SELECT count(*) FROM guests WHERE deleted_at IS NULL)`

func (r *Postgres) Stats(ctx context.Context) (*domain.Stats, error) {
	var s domain.Stats
	err := database.Conn(ctx, r.pool).QueryRow(ctx, statsSQL).Scan(
		&s.Users, &s.ActiveSubscriptions, &s.GraceSubscriptions, &s.PendingOrders,
		&s.Invitations, &s.PublishedInvitations, &s.RevenueThisMonth, &s.Guests)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
