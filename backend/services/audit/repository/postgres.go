package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/audit/domain"
	"undangan/kernel/database"
)

type Postgres struct{ pool *pgxpool.Pool }

var _ domain.Repository = (*Postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) Insert(ctx context.Context, l *domain.Log) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO audit_logs (actor_id, action, target, meta) VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		l.ActorID, l.Action, l.Target, l.Meta).Scan(&l.ID, &l.CreatedAt)
}

func (r *Postgres) List(ctx context.Context, limit, offset int) ([]domain.Log, int, error) {
	conn := database.Conn(ctx, r.pool)
	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM audit_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := conn.Query(ctx, `
		SELECT a.id, a.actor_id, u.name, a.action, a.target, a.meta, a.created_at
		FROM audit_logs a LEFT JOIN users u ON u.id = a.actor_id
		ORDER BY a.created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Log
	for rows.Next() {
		var l domain.Log
		if err := rows.Scan(&l.ID, &l.ActorID, &l.ActorName, &l.Action, &l.Target, &l.Meta, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}
