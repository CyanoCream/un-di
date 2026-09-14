-- Link tamu tervalidasi (/slug), mode akses, check-in venue (fitur paket), konfirmasi hadiah.

ALTER TABLE plans ADD COLUMN allow_checkin boolean NOT NULL DEFAULT false;
ALTER TABLE subscriptions ADD COLUMN allow_checkin boolean NOT NULL DEFAULT false;
UPDATE plans SET allow_checkin = true WHERE name = 'Paket Premium';
UPDATE subscriptions s SET allow_checkin = p.allow_checkin FROM plans p WHERE p.id = s.plan_id;

ALTER TABLE invitations
    ADD COLUMN access_mode text NOT NULL DEFAULT 'public' CHECK (access_mode IN ('public', 'guest_only')),
    ADD COLUMN checkin_enabled boolean NOT NULL DEFAULT false,
    ADD COLUMN checkin_pin_hash text;

ALTER TABLE guests
    ADD COLUMN slug text,
    ADD COLUMN checked_in_pax int,
    ADD COLUMN checked_in_by text NOT NULL DEFAULT '';

-- Isi slug untuk tamu lama; bentrok → tambahkan kode tamu.
UPDATE guests SET slug = COALESCE(NULLIF(trim(BOTH '-' FROM regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')), ''), 'tamu');
UPDATE guests g SET slug = g.slug || '-' || lower(g.code)
WHERE g.deleted_at IS NULL AND EXISTS (
    SELECT 1 FROM guests o WHERE o.invitation_id = g.invitation_id AND o.slug = g.slug AND o.id <> g.id AND o.deleted_at IS NULL);
UPDATE guests SET slug = slug || '-' || lower(code) WHERE deleted_at IS NOT NULL;
ALTER TABLE guests ALTER COLUMN slug SET NOT NULL;
CREATE UNIQUE INDEX guests_slug_uq ON guests (invitation_id, slug) WHERE deleted_at IS NULL;

CREATE TABLE checkin_logs (
    id            bigserial PRIMARY KEY,
    invitation_id uuid NOT NULL REFERENCES invitations (id) ON DELETE CASCADE,
    guest_id      uuid REFERENCES guests (id) ON DELETE SET NULL,
    input         text NOT NULL DEFAULT '',
    result        text NOT NULL CHECK (result IN ('checked_in', 'already', 'not_found', 'disabled', 'undo')),
    pax           int,
    station       text NOT NULL DEFAULT '',
    ip            text NOT NULL DEFAULT '',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX checkin_logs_invitation_idx ON checkin_logs (invitation_id, created_at DESC);

CREATE TABLE gift_confirmations (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id uuid NOT NULL REFERENCES invitations (id) ON DELETE CASCADE,
    guest_id      uuid REFERENCES guests (id) ON DELETE SET NULL,
    name          text NOT NULL,
    type          text NOT NULL CHECK (type IN ('transfer', 'kado')),
    account_label text NOT NULL DEFAULT '',
    amount        bigint CHECK (amount IS NULL OR amount >= 0),
    message       text NOT NULL DEFAULT '',
    proof_key     text NOT NULL,              -- kunci objek privat di storage
    ip            text NOT NULL DEFAULT '',
    is_verified   boolean NOT NULL DEFAULT false,
    verified_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE INDEX gift_confirmations_invitation_idx ON gift_confirmations (invitation_id, created_at DESC) WHERE deleted_at IS NULL;
