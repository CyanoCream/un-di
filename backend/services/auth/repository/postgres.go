package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/auth/domain"
	"undangan/kernel/database"
)

type Postgres struct{ pool *pgxpool.Pool }

var _ domain.RefreshTokenRepository = (*Postgres)(nil)

func NewPostgres(pool *pgxpool.Pool) *Postgres { return &Postgres{pool: pool} }

func (r *Postgres) Create(ctx context.Context, t *domain.RefreshToken) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO refresh_tokens (user_id, family_id, token_hash, portal, ip, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		t.UserID, t.FamilyID, t.TokenHash, t.Portal, t.IP, t.UserAgent, t.ExpiresAt).Scan(&t.ID)
}

func (r *Postgres) FindByHashForUpdate(ctx context.Context, hash []byte) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		SELECT id, user_id, family_id, token_hash, portal, ip, user_agent, expires_at, revoked_at
		FROM refresh_tokens WHERE token_hash = $1 FOR UPDATE`, hash).
		Scan(&t.ID, &t.UserID, &t.FamilyID, &t.TokenHash, &t.Portal, &t.IP, &t.UserAgent, &t.ExpiresAt, &t.RevokedAt)
	if err != nil {
		return nil, database.NotFound(err)
	}
	return &t, nil
}

func (r *Postgres) Revoke(ctx context.Context, id string) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

func (r *Postgres) RevokeFamily(ctx context.Context, familyID string) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now() WHERE family_id = $1 AND revoked_at IS NULL`, familyID)
	return err
}

func (r *Postgres) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func (r *Postgres) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `DELETE FROM refresh_tokens WHERE expires_at < $1`, before)
	return tag.RowsAffected(), err
}
