package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/invitation/domain"
	"undangan/kernel/database"
)

type Gifts struct{ pool *pgxpool.Pool }

var _ domain.GiftRepository = (*Gifts)(nil)

func NewGifts(pool *pgxpool.Pool) *Gifts { return &Gifts{pool: pool} }

const giftSelect = `
	SELECT gc.id, gc.invitation_id, gc.guest_id, g.name, gc.name, gc.type, gc.account_label, gc.amount, gc.message,
		gc.proof_key, gc.is_verified, gc.verified_at, gc.created_at
	FROM gift_confirmations gc LEFT JOIN guests g ON g.id = gc.guest_id`

func scanGift(row pgx.Row) (*domain.GiftConfirmation, error) {
	var g domain.GiftConfirmation
	if err := row.Scan(&g.ID, &g.InvitationID, &g.GuestID, &g.GuestName, &g.Name, &g.Type, &g.AccountLabel, &g.Amount, &g.Message,
		&g.ProofKey, &g.IsVerified, &g.VerifiedAt, &g.CreatedAt); err != nil {
		return nil, database.NotFound(err)
	}
	g.ProofURL = "/api/v1/invitations/" + g.InvitationID + "/gifts/" + g.ID + "/proof"
	return &g, nil
}

func (r *Gifts) Create(ctx context.Context, g *domain.GiftConfirmation) error {
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO gift_confirmations (invitation_id, guest_id, name, type, account_label, amount, message, proof_key, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at`,
		g.InvitationID, g.GuestID, g.Name, g.Type, g.AccountLabel, g.Amount, g.Message, g.ProofKey, g.IP).Scan(&g.ID, &g.CreatedAt)
	g.ProofURL = "/api/v1/invitations/" + g.InvitationID + "/gifts/" + g.ID + "/proof"
	return err
}

func (r *Gifts) List(ctx context.Context, invID string, limit, offset int) ([]domain.GiftConfirmation, int, error) {
	conn := database.Conn(ctx, r.pool)
	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM gift_confirmations WHERE invitation_id = $1 AND deleted_at IS NULL`, invID).Scan(&total); err != nil {
		return nil, 0, database.NotFound(err)
	}
	if limit <= 0 {
		limit = 100000
	}
	rows, err := conn.Query(ctx, giftSelect+` WHERE gc.invitation_id = $1 AND gc.deleted_at IS NULL
		ORDER BY gc.created_at DESC LIMIT $2 OFFSET $3`, invID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.GiftConfirmation
	for rows.Next() {
		g, err := scanGift(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *g)
	}
	return out, total, rows.Err()
}

func (r *Gifts) Summary(ctx context.Context, invID string) (domain.GiftSummary, error) {
	var s domain.GiftSummary
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE is_verified),
			COALESCE(sum(amount), 0), COALESCE(sum(amount) FILTER (WHERE is_verified), 0)
		FROM gift_confirmations WHERE invitation_id = $1 AND deleted_at IS NULL`, invID).
		Scan(&s.Count, &s.VerifiedCount, &s.TotalAmount, &s.VerifiedAmount)
	return s, database.NotFound(err)
}

func (r *Gifts) FindByID(ctx context.Context, invID, id string) (*domain.GiftConfirmation, error) {
	return scanGift(database.Conn(ctx, r.pool).QueryRow(ctx, giftSelect+`
		WHERE gc.id = $1 AND gc.invitation_id = $2 AND gc.deleted_at IS NULL`, id, invID))
}

func (r *Gifts) SetVerified(ctx context.Context, invID, id string, verified bool) (*domain.GiftConfirmation, error) {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `
		UPDATE gift_confirmations SET is_verified = $3, verified_at = CASE WHEN $3 THEN now() ELSE NULL END
		WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, id, invID, verified)
	if err := affected(database.NotFound(err), tag.RowsAffected()); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, invID, id)
}

func (r *Gifts) SoftDelete(ctx context.Context, invID, id string) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx,
		`UPDATE gift_confirmations SET deleted_at = now() WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, id, invID)
	return affected(database.NotFound(err), tag.RowsAffected())
}
