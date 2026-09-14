package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

type Guests struct{ pool *pgxpool.Pool }

var _ domain.GuestRepository = (*Guests)(nil)

func NewGuests(pool *pgxpool.Pool) *Guests { return &Guests{pool: pool} }

const guestCols = `id, invitation_id, name, phone, group_name, pax, code, slug, opened_at, checked_in_at, checked_in_pax, checked_in_by, created_at`

func scanGuest(row pgx.Row) (*domain.Guest, error) {
	var g domain.Guest
	if err := row.Scan(&g.ID, &g.InvitationID, &g.Name, &g.Phone, &g.GroupName, &g.Pax, &g.Code, &g.Slug, &g.OpenedAt, &g.CheckedInAt, &g.CheckedInPax, &g.CheckedInBy, &g.CreatedAt); err != nil {
		return nil, database.NotFound(err)
	}
	return &g, nil
}

func collectGuests(rows pgx.Rows) ([]domain.Guest, error) {
	defer rows.Close()
	var out []domain.Guest
	for rows.Next() {
		g, err := scanGuest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func (r *Guests) List(ctx context.Context, invID string, f domain.GuestFilter) ([]domain.Guest, int, error) {
	const where = ` WHERE invitation_id = $1 AND deleted_at IS NULL
		AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR phone LIKE '%' || $2 || '%' OR code = upper($2))
		AND ($3 = '' OR group_name = $3)`
	conn := database.Conn(ctx, r.pool)
	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM guests`+where, invID, f.Query, f.Group).Scan(&total); err != nil {
		return nil, 0, database.NotFound(err)
	}
	rows, err := conn.Query(ctx, `SELECT `+guestCols+` FROM guests`+where+` ORDER BY created_at DESC, name LIMIT $4 OFFSET $5`,
		invID, f.Query, f.Group, f.Limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	out, err := collectGuests(rows)
	return out, total, err
}

func (r *Guests) ListAll(ctx context.Context, invID string) ([]domain.Guest, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx,
		`SELECT `+guestCols+` FROM guests WHERE invitation_id = $1 AND deleted_at IS NULL ORDER BY group_name, name`, invID)
	if err != nil {
		return nil, err
	}
	return collectGuests(rows)
}

func (r *Guests) Count(ctx context.Context, invID string) (int, error) {
	var n int
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT count(*) FROM guests WHERE invitation_id = $1 AND deleted_at IS NULL`, invID).Scan(&n)
	return n, err
}

func (r *Guests) Create(ctx context.Context, g *domain.Guest) error {
	err := database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO guests (invitation_id, name, phone, group_name, pax, code, slug) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`, g.InvitationID, g.Name, g.Phone, g.GroupName, g.Pax, g.Code, g.Slug).Scan(&g.ID, &g.CreatedAt)
	if database.IsUniqueViolation(err) {
		return apperror.Conflict("code_taken", "Kode atau slug tamu bentrok")
	}
	return err
}

func (r *Guests) FindByID(ctx context.Context, invID, id string) (*domain.Guest, error) {
	return scanGuest(database.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT `+guestCols+` FROM guests WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, id, invID))
}

func (r *Guests) FindByCode(ctx context.Context, invID, code string) (*domain.Guest, error) {
	return scanGuest(database.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT `+guestCols+` FROM guests WHERE invitation_id = $1 AND code = $2 AND deleted_at IS NULL`, invID, code))
}

func (r *Guests) FindBySlug(ctx context.Context, invID, slug string) (*domain.Guest, error) {
	return scanGuest(database.Conn(ctx, r.pool).QueryRow(ctx,
		`SELECT `+guestCols+` FROM guests WHERE invitation_id = $1 AND slug = lower($2) AND deleted_at IS NULL`, invID, slug))
}

func (r *Guests) TakenSlugs(ctx context.Context, invID string) (map[string]bool, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `SELECT slug FROM guests WHERE invitation_id = $1 AND deleted_at IS NULL`, invID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out[s] = true
	}
	return out, rows.Err()
}

func (r *Guests) Search(ctx context.Context, invID, q string, limit int) ([]domain.Guest, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `
		SELECT `+guestCols+` FROM guests
		WHERE invitation_id = $1 AND deleted_at IS NULL
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR code = upper($2) OR group_name ILIKE '%' || $2 || '%')
		ORDER BY (checked_in_at IS NOT NULL), name LIMIT $3`, invID, q, limit)
	if err != nil {
		return nil, err
	}
	return collectGuests(rows)
}

func (r *Guests) CheckIn(ctx context.Context, invID, id string, pax int, by string) (*domain.Guest, bool, error) {
	g, err := scanGuest(database.Conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE guests SET checked_in_at = now(), checked_in_pax = $3, checked_in_by = $4, updated_at = now()
		WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL AND checked_in_at IS NULL
		RETURNING `+guestCols, id, invID, pax, by))
	if err == nil {
		return g, true, nil
	}
	if !errors.Is(err, apperror.ErrNotFound) {
		return nil, false, err
	}
	// Tidak ter-update: sudah check-in, atau tamu tidak ada.
	g, err = r.FindByID(ctx, invID, id)
	return g, false, err
}

func (r *Guests) UndoCheckIn(ctx context.Context, invID, id string) (*domain.Guest, error) {
	return scanGuest(database.Conn(ctx, r.pool).QueryRow(ctx, `
		UPDATE guests SET checked_in_at = NULL, checked_in_pax = NULL, checked_in_by = '', updated_at = now()
		WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL
		RETURNING `+guestCols, id, invID))
}

func (r *Guests) Attendance(ctx context.Context, invID string, f domain.AttendanceFilter) ([]domain.AttendanceItem, int, error) {
	const where = ` WHERE g.invitation_id = $1 AND g.deleted_at IS NULL
		AND ($2 = '' OR g.name ILIKE '%' || $2 || '%' OR g.code = upper($2) OR g.group_name ILIKE '%' || $2 || '%')
		AND ($3 = 'all' OR ($3 = 'checked_in' AND g.checked_in_at IS NOT NULL) OR ($3 = 'not_checked_in' AND g.checked_in_at IS NULL))`
	status := f.Status
	if status != "checked_in" && status != "not_checked_in" {
		status = "all"
	}
	conn := database.Conn(ctx, r.pool)
	var total int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM guests g`+where, invID, f.Query, status).Scan(&total); err != nil {
		return nil, 0, database.NotFound(err)
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 100000
	}
	rows, err := conn.Query(ctx, `
		SELECT g.id, g.name, g.group_name, g.phone, g.pax, g.code, w.attendance, w.pax,
			g.checked_in_at, g.checked_in_pax, g.checked_in_by
		FROM guests g
		LEFT JOIN LATERAL (
			SELECT attendance::text AS attendance, pax FROM wishes
			WHERE guest_id = g.id AND deleted_at IS NULL ORDER BY created_at DESC LIMIT 1
		) w ON true`+where+`
		ORDER BY g.checked_in_at DESC NULLS LAST, g.group_name, g.name LIMIT $4 OFFSET $5`,
		invID, f.Query, status, limit, f.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []domain.AttendanceItem
	for rows.Next() {
		var it domain.AttendanceItem
		if err := rows.Scan(&it.GuestID, &it.Name, &it.GroupName, &it.Phone, &it.Pax, &it.Code, &it.RSVPAttendance, &it.RSVPPax,
			&it.CheckedInAt, &it.CheckedInPax, &it.CheckedInBy); err != nil {
			return nil, 0, err
		}
		out = append(out, it)
	}
	return out, total, rows.Err()
}

func (r *Guests) AttendanceSummary(ctx context.Context, invID string) (domain.AttendanceSummary, error) {
	var s domain.AttendanceSummary
	conn := database.Conn(ctx, r.pool)
	err := conn.QueryRow(ctx, `
		SELECT count(*), COALESCE(sum(pax), 0),
			count(*) FILTER (WHERE checked_in_at IS NOT NULL),
			COALESCE(sum(checked_in_pax) FILTER (WHERE checked_in_at IS NOT NULL), 0)
		FROM guests WHERE invitation_id = $1 AND deleted_at IS NULL`, invID).
		Scan(&s.InvitedGuests, &s.InvitedPax, &s.CheckedInGuests, &s.CheckedInPax)
	if err != nil {
		return s, database.NotFound(err)
	}
	s.NotCheckedInGuests = s.InvitedGuests - s.CheckedInGuests
	err = conn.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE attendance = 'hadir'), COALESCE(sum(pax) FILTER (WHERE attendance = 'hadir'), 0)
		FROM wishes WHERE invitation_id = $1 AND deleted_at IS NULL`, invID).Scan(&s.RSVPHadir, &s.RSVPPax)
	return s, database.NotFound(err)
}

func (r *Guests) Update(ctx context.Context, g *domain.Guest) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `
		UPDATE guests SET name = $3, phone = $4, group_name = $5, pax = $6, updated_at = now()
		WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, g.ID, g.InvitationID, g.Name, g.Phone, g.GroupName, g.Pax)
	return affected(err, tag.RowsAffected())
}

func (r *Guests) SoftDelete(ctx context.Context, invID, id string) error {
	tag, err := database.Conn(ctx, r.pool).Exec(ctx,
		`UPDATE guests SET deleted_at = now() WHERE id = $1 AND invitation_id = $2 AND deleted_at IS NULL`, id, invID)
	if err != nil {
		return database.NotFound(err)
	}
	return affected(nil, tag.RowsAffected())
}

func (r *Guests) ExistingKeys(ctx context.Context, invID string) (map[string]bool, map[string]bool, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx,
		`SELECT name, phone, code, deleted_at IS NULL FROM guests WHERE invitation_id = $1`, invID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	keys, codes := map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var name, phone, code string
		var active bool
		if err := rows.Scan(&name, &phone, &code, &active); err != nil {
			return nil, nil, err
		}
		codes[code] = true // unique index kode mencakup baris terhapus
		if active {
			keys[domain.DedupKey(name, phone)] = true
		}
	}
	return keys, codes, rows.Err()
}

func (r *Guests) BulkInsert(ctx context.Context, guests []domain.Guest) (int, error) {
	n, err := database.Conn(ctx, r.pool).CopyFrom(ctx, pgx.Identifier{"guests"},
		[]string{"invitation_id", "name", "phone", "group_name", "pax", "code", "slug"},
		pgx.CopyFromSlice(len(guests), func(i int) ([]any, error) {
			g := guests[i]
			return []any{g.InvitationID, g.Name, g.Phone, g.GroupName, g.Pax, g.Code, g.Slug}, nil
		}))
	return int(n), err
}

func (r *Guests) MarkOpened(ctx context.Context, id string) error {
	_, err := database.Conn(ctx, r.pool).Exec(ctx, `UPDATE guests SET opened_at = now() WHERE id = $1 AND opened_at IS NULL`, id)
	return err
}
