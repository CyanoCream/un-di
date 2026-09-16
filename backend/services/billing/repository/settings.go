package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/database"
	"undangan/services/billing/domain"
)

type Settings struct{ pool *pgxpool.Pool }

var _ domain.SettingsRepository = (*Settings)(nil)

func NewSettings(pool *pgxpool.Pool) *Settings { return &Settings{pool: pool} }

func (r *Settings) GetPayment(ctx context.Context) (*domain.PaymentSettings, error) {
	var raw []byte
	if err := database.Conn(ctx, r.pool).QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`,
		domain.PaymentSettingsKey).Scan(&raw); err != nil {
		return nil, database.NotFound(err)
	}
	var s domain.PaymentSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Settings) SavePayment(ctx context.Context, s *domain.PaymentSettings) error {
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = database.Conn(ctx, r.pool).Exec(ctx, `
		INSERT INTO settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		domain.PaymentSettingsKey, string(raw))
	return err
}

type Notifications struct{ pool *pgxpool.Pool }

var _ domain.NotificationRepository = (*Notifications)(nil)

func NewNotifications(pool *pgxpool.Pool) *Notifications { return &Notifications{pool: pool} }

func (r *Notifications) Insert(ctx context.Context, n domain.Notification) (bool, error) {
	if n.Channel == "" {
		n.Channel = domain.ChannelInApp
	}
	var subID *string
	if n.SubscriptionID != "" {
		subID = &n.SubscriptionID
	}
	tag, err := database.Conn(ctx, r.pool).Exec(ctx, `
		INSERT INTO notifications (user_id, subscription_id, kind, channel, ref) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT DO NOTHING`, n.UserID, subID, n.Kind, n.Channel, n.Ref)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
