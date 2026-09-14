package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/billing/domain"
	"undangan/kernel/database"
)

type Plans struct{ pool *pgxpool.Pool }

var _ domain.PlanRepository = (*Plans)(nil)

func NewPlans(pool *pgxpool.Pool) *Plans { return &Plans{pool: pool} }

const planCols = `id, name, description, price, duration_days, grace_days, max_invitations, max_guests,
	allow_custom_domain, allow_checkin, is_active, sort_order, created_at`

func scanPlan(row pgx.Row) (*domain.Plan, error) {
	var p domain.Plan
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.DurationDays, &p.GraceDays, &p.MaxInvitations,
		&p.MaxGuests, &p.AllowCustomDomain, &p.AllowCheckin, &p.IsActive, &p.SortOrder, &p.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &p, nil
}

func (r *Plans) Create(ctx context.Context, p *domain.Plan) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO plans (name, description, price, duration_days, grace_days, max_invitations, max_guests,
			allow_custom_domain, allow_checkin, is_active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`,
		p.Name, p.Description, p.Price, p.DurationDays, p.GraceDays, p.MaxInvitations, p.MaxGuests,
		p.AllowCustomDomain, p.AllowCheckin, p.IsActive, p.SortOrder).Scan(&p.ID, &p.CreatedAt)
}

func (r *Plans) Update(ctx context.Context, p *domain.Plan) error {
	return execOne(ctx, r.pool, `
		UPDATE plans SET name = $2, description = $3, price = $4, duration_days = $5, grace_days = $6,
			max_invitations = $7, max_guests = $8, allow_custom_domain = $9, is_active = $10, sort_order = $11, allow_checkin = $12
		WHERE id = $1`,
		p.ID, p.Name, p.Description, p.Price, p.DurationDays, p.GraceDays, p.MaxInvitations, p.MaxGuests,
		p.AllowCustomDomain, p.IsActive, p.SortOrder, p.AllowCheckin)
}

func (r *Plans) FindByID(ctx context.Context, id string) (*domain.Plan, error) {
	return scanPlan(database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT `+planCols+` FROM plans WHERE id = $1`, id))
}

func (r *Plans) List(ctx context.Context, activeOnly bool) ([]domain.Plan, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `SELECT `+planCols+` FROM plans
		WHERE (NOT $1::boolean OR is_active) ORDER BY sort_order, price, created_at`, activeOnly)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Plan
	for rows.Next() {
		p, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *Plans) Count(ctx context.Context) (int, error) {
	var n int
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM plans`).Scan(&n)
	return n, err
}

func (r *Plans) MinActiveMaxGuests(ctx context.Context) (int, bool, error) {
	var n *int
	if err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT min(max_guests) FROM plans WHERE is_active`).Scan(&n); err != nil {
		return 0, false, err
	}
	if n == nil {
		return 0, false, nil
	}
	return *n, true, nil
}
