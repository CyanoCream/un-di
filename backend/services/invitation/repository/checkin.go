package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/invitation/domain"
	"undangan/kernel/database"
)

type CheckinLogs struct{ pool *pgxpool.Pool }

var _ domain.CheckinLogRepository = (*CheckinLogs)(nil)

func NewCheckinLogs(pool *pgxpool.Pool) *CheckinLogs { return &CheckinLogs{pool: pool} }

func (r *CheckinLogs) Insert(ctx context.Context, l *domain.CheckinLog) error {
	return database.Conn(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO checkin_logs (invitation_id, guest_id, input, result, pax, station, ip)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, created_at`,
		l.InvitationID, l.GuestID, l.Input, l.Result, l.Pax, l.Station, l.IP).Scan(&l.ID, &l.CreatedAt)
}

func (r *CheckinLogs) Recent(ctx context.Context, invID string, limit int) ([]domain.CheckinLog, error) {
	rows, err := database.Conn(ctx, r.pool).Query(ctx, `
		SELECT l.id, l.guest_id, g.name, l.input, l.result, l.pax, l.station, l.created_at
		FROM checkin_logs l LEFT JOIN guests g ON g.id = l.guest_id
		WHERE l.invitation_id = $1 ORDER BY l.created_at DESC LIMIT $2`, invID, limit)
	if err != nil {
		return nil, database.NotFound(err)
	}
	defer rows.Close()
	var out []domain.CheckinLog
	for rows.Next() {
		var l domain.CheckinLog
		if err := rows.Scan(&l.ID, &l.GuestID, &l.GuestName, &l.Input, &l.Result, &l.Pax, &l.Station, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
