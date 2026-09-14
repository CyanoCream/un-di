// Package repository (invitation) = implementasi Postgres untuk repository undangan, tamu, dan ucapan.
package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

type Invitations struct{ pool *pgxpool.Pool }

var _ domain.InvitationRepository = (*Invitations)(nil)

func NewInvitations(pool *pgxpool.Pool) *Invitations { return &Invitations{pool: pool} }

const invSelect = `
	SELECT i.id, i.user_id, u.name, u.email, i.theme, i.theme_locked_at, i.status, i.content,
		(SELECT d.hostname FROM domains d WHERE d.invitation_id = i.id AND d.kind = 'subdomain' AND d.deleted_at IS NULL LIMIT 1),
		(SELECT count(*) FROM guests g WHERE g.invitation_id = i.id AND g.deleted_at IS NULL),
		i.published_at, i.suspended_at, i.created_at, i.updated_at,
		i.access_mode, i.checkin_enabled, COALESCE(i.checkin_pin_hash, ''),
		cd.hostname, cd.verify_token, cd.verified_at
	FROM invitations i
	LEFT JOIN domains cd ON cd.invitation_id = i.id AND cd.kind = 'custom' AND cd.deleted_at IS NULL JOIN users u ON u.id = i.user_id`

func scanInvitation(row pgx.Row) (*domain.Invitation, error) {
	var inv domain.Invitation
	var cdHost, cdToken *string
	var cdVerified *time.Time
	err := row.Scan(&inv.ID, &inv.UserID, &inv.OwnerName, &inv.OwnerEmail, &inv.Theme, &inv.ThemeLockedAt, &inv.Status,
		&inv.Content, &inv.Subdomain, &inv.GuestCount, &inv.PublishedAt, &inv.SuspendedAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.AccessMode, &inv.CheckinEnabled, &inv.CheckinPinHash, &cdHost, &cdToken, &cdVerified)
	if err != nil {
		return nil, database.NotFound(err)
	}
	if cdHost != nil {
		inv.CustomDomain = &domain.CustomDomain{Hostname: *cdHost, VerifiedAt: cdVerified}
		if cdToken != nil {
			inv.CustomDomain.VerifyToken = *cdToken
		}
	}
	inv.Content = domain.NormalizeContent(inv.Content)
	return &inv, nil
}

func (r *Invitations) conn(ctx context.Context) database.DBTX { return database.Conn(ctx, r.pool) }

func affected(tagErr error, n int64) error {
	if tagErr != nil {
		return tagErr
	}
	if n == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

func (r *Invitations) Create(ctx context.Context, inv *domain.Invitation) error {
	err := r.conn(ctx).QueryRow(ctx, `
		INSERT INTO invitations (user_id, status, content) VALUES ($1, $2, $3) RETURNING id, created_at, updated_at`,
		inv.UserID, inv.Status, inv.Content).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if database.IsForeignKeyViolation(err) {
		return apperror.Validation(map[string]string{"user_id": "User tidak ditemukan"})
	}
	return err
}

func (r *Invitations) FindByID(ctx context.Context, id string) (*domain.Invitation, error) {
	return scanInvitation(r.conn(ctx).QueryRow(ctx, invSelect+` WHERE i.id = $1 AND i.deleted_at IS NULL`, id))
}

func (r *Invitations) LockForUpdate(ctx context.Context, id string) error {
	var x string
	return database.NotFound(r.conn(ctx).QueryRow(ctx, `SELECT id FROM invitations WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&x))
}

func (r *Invitations) List(ctx context.Context, f domain.ListFilter) ([]domain.Invitation, int, error) {
	const where = ` WHERE i.deleted_at IS NULL
		AND ($1 = '' OR i.user_id::text = $1)
		AND ($2 = '' OR i.status::text = $2)
		AND ($3 = '' OR u.name ILIKE '%' || $3 || '%' OR u.email ILIKE '%' || $3 || '%'
			OR i.content->'groom'->>'nickname' ILIKE '%' || $3 || '%' OR i.content->'bride'->>'nickname' ILIKE '%' || $3 || '%'
			OR i.content->'groom'->>'full_name' ILIKE '%' || $3 || '%' OR i.content->'bride'->>'full_name' ILIKE '%' || $3 || '%'
			OR EXISTS (SELECT 1 FROM domains d WHERE d.invitation_id = i.id AND d.deleted_at IS NULL AND d.hostname ILIKE '%' || $3 || '%'))`
	var total int
	if err := r.conn(ctx).QueryRow(ctx, `SELECT count(*) FROM invitations i JOIN users u ON u.id = i.user_id`+where,
		f.UserID, f.Status, f.Query).Scan(&total); err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.conn(ctx).Query(ctx, invSelect+where+` ORDER BY i.updated_at DESC LIMIT $4 OFFSET $5`,
		f.UserID, f.Status, f.Query, limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.Invitation
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *inv)
	}
	return out, total, rows.Err()
}

func (r *Invitations) CountByUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.conn(ctx).QueryRow(ctx, `SELECT count(*) FROM invitations WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&n)
	return n, err
}

func (r *Invitations) UpdateContent(ctx context.Context, id string, c domain.Content) error {
	tag, err := r.conn(ctx).Exec(ctx, `UPDATE invitations SET content = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, c)
	return affected(err, tag.RowsAffected())
}

func (r *Invitations) SetTheme(ctx context.Context, id, theme string) error {
	tag, err := r.conn(ctx).Exec(ctx, `
		UPDATE invitations SET theme = $2, theme_locked_at = COALESCE(theme_locked_at, now()), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id, theme)
	return affected(err, tag.RowsAffected())
}

func (r *Invitations) SetStatus(ctx context.Context, id string, status domain.Status) error {
	tag, err := r.conn(ctx).Exec(ctx, `
		UPDATE invitations SET status = $2::text::invitation_status, updated_at = now(),
			published_at = CASE WHEN $2::text = 'published' THEN COALESCE(published_at, now()) ELSE published_at END
		WHERE id = $1 AND deleted_at IS NULL`, id, status)
	return affected(err, tag.RowsAffected())
}

func (r *Invitations) UpdateSettings(ctx context.Context, id, accessMode string, checkinEnabled bool, pinHash *string) error {
	tag, err := r.conn(ctx).Exec(ctx, `
		UPDATE invitations SET access_mode = $2, checkin_enabled = $3,
			checkin_pin_hash = COALESCE($4, checkin_pin_hash), updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`, id, accessMode, checkinEnabled, pinHash)
	return affected(err, tag.RowsAffected())
}

func (r *Invitations) SoftDelete(ctx context.Context, id string) error {
	conn := r.conn(ctx)
	tag, err := conn.Exec(ctx, `UPDATE invitations SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err := affected(database.NotFound(err), tag.RowsAffected()); err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `UPDATE domains SET deleted_at = now() WHERE invitation_id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *Invitations) SubdomainOwner(ctx context.Context, label string) (string, error) {
	var id string
	err := r.conn(ctx).QueryRow(ctx, `SELECT invitation_id FROM domains WHERE lower(hostname) = lower($1) AND deleted_at IS NULL`, label).Scan(&id)
	return id, database.NotFound(err)
}

func (r *Invitations) ReplaceSubdomain(ctx context.Context, invitationID, label string) error {
	conn := r.conn(ctx)
	if _, err := conn.Exec(ctx, `UPDATE domains SET deleted_at = now() WHERE invitation_id = $1 AND kind = 'subdomain' AND deleted_at IS NULL`, invitationID); err != nil {
		return err
	}
	_, err := conn.Exec(ctx, `INSERT INTO domains (invitation_id, kind, hostname) VALUES ($1, 'subdomain', $2)`, invitationID, label)
	if database.IsUniqueViolation(err) {
		return apperror.Conflict("subdomain_taken", "Subdomain sudah dipakai")
	}
	if err == nil {
		_, err = conn.Exec(ctx, `UPDATE invitations SET updated_at = now() WHERE id = $1`, invitationID)
	}
	return err
}

func (r *Invitations) ReplaceCustomDomain(ctx context.Context, invID, hostname, token string) error {
	conn := r.conn(ctx)
	if _, err := conn.Exec(ctx, `UPDATE domains SET deleted_at = now() WHERE invitation_id = $1 AND kind = 'custom' AND deleted_at IS NULL`, invID); err != nil {
		return err
	}
	_, err := conn.Exec(ctx, `INSERT INTO domains (invitation_id, kind, hostname, verify_token) VALUES ($1, 'custom', $2, $3)`, invID, hostname, token)
	if database.IsUniqueViolation(err) {
		return apperror.Conflict("domain_taken", "Domain sudah terhubung ke undangan lain")
	}
	return err
}

func (r *Invitations) MarkCustomDomainVerified(ctx context.Context, invID string) error {
	tag, err := r.conn(ctx).Exec(ctx, `
		UPDATE domains SET verified_at = now() WHERE invitation_id = $1 AND kind = 'custom' AND deleted_at IS NULL`, invID)
	return affected(err, tag.RowsAffected())
}

func (r *Invitations) RemoveCustomDomain(ctx context.Context, invID string) error {
	_, err := r.conn(ctx).Exec(ctx, `UPDATE domains SET deleted_at = now() WHERE invitation_id = $1 AND kind = 'custom' AND deleted_at IS NULL`, invID)
	return err
}

func (r *Invitations) FindPublishedBySubdomain(ctx context.Context, label string) (*domain.Invitation, error) {
	return scanInvitation(r.conn(ctx).QueryRow(ctx, invSelect+`
		JOIN domains dm ON dm.invitation_id = i.id AND dm.kind = 'subdomain' AND dm.deleted_at IS NULL
		WHERE lower(dm.hostname) = lower($1) AND i.status = 'published' AND i.deleted_at IS NULL`, label))
}

func (r *Invitations) FindPublishedByCustomDomain(ctx context.Context, hostname string) (*domain.Invitation, error) {
	return scanInvitation(r.conn(ctx).QueryRow(ctx, invSelect+`
		JOIN domains dm ON dm.invitation_id = i.id AND dm.kind = 'custom' AND dm.deleted_at IS NULL AND dm.verified_at IS NOT NULL
		WHERE lower(dm.hostname) = lower($1) AND i.status = 'published' AND i.deleted_at IS NULL`, hostname))
}

func (r *Invitations) FindPublishedByID(ctx context.Context, id string) (*domain.Invitation, error) {
	return scanInvitation(r.conn(ctx).QueryRow(ctx, invSelect+` WHERE i.id = $1 AND i.status = 'published' AND i.deleted_at IS NULL`, id))
}

func (r *Invitations) IsVerifiedCustomDomain(ctx context.Context, hostname string) (bool, error) {
	var ok bool
	err := r.conn(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM domains d JOIN invitations i ON i.id = d.invitation_id
		WHERE d.kind = 'custom' AND lower(d.hostname) = lower($1) AND d.verified_at IS NOT NULL
		  AND d.deleted_at IS NULL AND i.deleted_at IS NULL)`, hostname).Scan(&ok)
	return ok, err
}

// SuspendForUser: undangan → suspended + soft delete; domain dilepas dengan deleted_at = suspended_at
// (penanda supaya RestoreForUser hanya memulihkan domain yang dilepas oleh proses ini).
func (r *Invitations) SuspendForUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.conn(ctx).QueryRow(ctx, `
		WITH s AS (
			UPDATE invitations SET status = 'suspended', suspended_at = now(), deleted_at = now(), updated_at = now()
			WHERE user_id = $1 AND deleted_at IS NULL
			RETURNING id, suspended_at
		), d AS (
			UPDATE domains dm SET deleted_at = s.suspended_at FROM s
			WHERE dm.invitation_id = s.id AND dm.deleted_at IS NULL
			RETURNING dm.id
		)
		SELECT (SELECT count(*) FROM s) + 0 * (SELECT count(*) FROM d)`, userID).Scan(&n)
	return n, err
}

func (r *Invitations) RestoreForUser(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.conn(ctx).QueryRow(ctx, `
		WITH inv AS (
			SELECT id, suspended_at, published_at FROM invitations
			WHERE user_id = $1 AND suspended_at IS NOT NULL AND deleted_at IS NOT NULL
		), dom AS (
			UPDATE domains d SET deleted_at = NULL FROM inv
			WHERE d.invitation_id = inv.id AND d.deleted_at = inv.suspended_at
			  AND NOT EXISTS (SELECT 1 FROM domains o WHERE lower(o.hostname) = lower(d.hostname) AND o.deleted_at IS NULL)
			RETURNING d.invitation_id, d.kind
		), upd AS (
			UPDATE invitations i SET deleted_at = NULL, suspended_at = NULL, updated_at = now(),
				status = CASE WHEN inv.published_at IS NOT NULL
					AND EXISTS (SELECT 1 FROM dom WHERE dom.invitation_id = i.id AND dom.kind = 'subdomain')
					THEN 'published'::invitation_status ELSE 'draft'::invitation_status END
			FROM inv WHERE i.id = inv.id
			RETURNING i.id
		)
		SELECT count(*) FROM upd`, userID).Scan(&n)
	return n, err
}

func (r *Invitations) PurgePersonalData(ctx context.Context, before time.Time) (int, error) {
	conn := r.conn(ctx)
	g, err := conn.Exec(ctx, `DELETE FROM guests WHERE invitation_id IN (SELECT id FROM invitations WHERE suspended_at < $1)`, before)
	if err != nil {
		return 0, err
	}
	w, err := conn.Exec(ctx, `DELETE FROM wishes WHERE invitation_id IN (SELECT id FROM invitations WHERE suspended_at < $1)`, before)
	if err != nil {
		return 0, err
	}
	// Log check-in & konfirmasi hadiah juga data pribadi. (File bukti di storage: TODO job pembersihan objek.)
	c, err := conn.Exec(ctx, `DELETE FROM checkin_logs WHERE invitation_id IN (SELECT id FROM invitations WHERE suspended_at < $1)`, before)
	if err != nil {
		return 0, err
	}
	gc, err := conn.Exec(ctx, `DELETE FROM gift_confirmations WHERE invitation_id IN (SELECT id FROM invitations WHERE suspended_at < $1)`, before)
	if err != nil {
		return 0, err
	}
	return int(g.RowsAffected() + w.RowsAffected() + c.RowsAffected() + gc.RowsAffected()), nil
}
