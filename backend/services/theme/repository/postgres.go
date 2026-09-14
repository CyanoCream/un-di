package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/theme/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

type Postgres struct{ pool *pgxpool.Pool }

var _ domain.Repository = (*Postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (r *Postgres) Upsert(ctx context.Context, m domain.Meta, defaultSort int) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `
		INSERT INTO themes (slug, name, description, tags, colors, fonts, sort_order, category, thumbnail)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description,
			tags = EXCLUDED.tags, colors = EXCLUDED.colors, fonts = EXCLUDED.fonts,
			category = EXCLUDED.category, thumbnail = EXCLUDED.thumbnail`,
		m.Slug, m.Name, m.Description, nonNil(m.Tags), nonNil(m.Colors), nonNil(m.Fonts), defaultSort, m.Category, m.Thumbnail)
	return err
}

func (r *Postgres) DeactivateMissing(ctx context.Context, present []string) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE themes SET is_active = false WHERE NOT (slug = ANY($1))`, nonNil(present))
	return err
}

const cols = `slug, name, description, tags, colors, fonts, is_active, is_premium, sort_order, category, thumbnail`

func scan(row pgx.Row) (*domain.Theme, error) {
	var t domain.Theme
	if err := row.Scan(&t.Slug, &t.Name, &t.Description, &t.Tags, &t.Colors, &t.Fonts, &t.IsActive, &t.IsPremium, &t.SortOrder, &t.Category, &t.Thumbnail); err != nil {
		return nil, database.NotFound(err)
	}
	t.PreviewURL = "/_preview/" + t.Slug
	return &t, nil
}

func (r *Postgres) List(ctx context.Context, onlyActive bool) ([]domain.Theme, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx,
		`SELECT `+cols+` FROM themes WHERE ($1 = false OR is_active) ORDER BY sort_order, name`, onlyActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Theme
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (r *Postgres) Get(ctx context.Context, slug string) (*domain.Theme, error) {
	return scan(database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT `+cols+` FROM themes WHERE slug = $1`, slug))
}

func (r *Postgres) Update(ctx context.Context, slug string, in domain.UpdateInput) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `
		UPDATE themes SET is_active = COALESCE($2, is_active), is_premium = COALESCE($3, is_premium),
			sort_order = COALESCE($4, sort_order) WHERE slug = $1`, slug, in.IsActive, in.IsPremium, in.SortOrder)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}
