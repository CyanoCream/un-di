-- Billing: notifikasi lifecycle diikat ke periode langganan.
-- Index lama (subscription_id, kind, channel) membuat pengingat tidak pernah terkirim lagi
-- setelah langganan diperpanjang. ref = akhir periode (RFC3339 UTC) saat notifikasi dibuat.

ALTER TABLE notifications ADD COLUMN ref text NOT NULL DEFAULT '';

DROP INDEX notifications_once_uq;
CREATE UNIQUE INDEX notifications_once_uq ON notifications (subscription_id, kind, channel, ref);
CREATE INDEX notifications_user_idx ON notifications (user_id, created_at DESC);
