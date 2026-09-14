-- Skema awal. Data customer memakai soft delete (deleted_at).

CREATE TYPE user_role AS ENUM ('super_admin', 'customer');
CREATE TYPE subscription_status AS ENUM ('active', 'grace', 'expired', 'cancelled');
CREATE TYPE order_status AS ENUM ('awaiting_payment', 'awaiting_confirmation', 'paid', 'rejected', 'expired');
CREATE TYPE invitation_status AS ENUM ('draft', 'published', 'suspended');
CREATE TYPE domain_kind AS ENUM ('subdomain', 'custom');
CREATE TYPE attendance AS ENUM ('hadir', 'tidak', 'ragu');

CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text NOT NULL,
    phone         text NOT NULL DEFAULT '',
    name          text NOT NULL,
    password_hash text NOT NULL,
    role          user_role NOT NULL DEFAULT 'customer',
    is_suspended  boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX users_email_uq ON users (lower(email)) WHERE deleted_at IS NULL;

-- Refresh token (opaque) untuk memperbarui access token JWT.
-- Dirotasi setiap dipakai; token lama yang dipakai ulang → seluruh family dicabut (indikasi pencurian).
CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family_id   uuid NOT NULL,
    token_hash  bytea NOT NULL UNIQUE,      -- sha256(token); token asli hanya di cookie
    portal      text NOT NULL,              -- admin | customer
    ip          text NOT NULL DEFAULT '',
    user_agent  text NOT NULL DEFAULT '',
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);

CREATE TABLE plans (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                text NOT NULL,
    description         text NOT NULL DEFAULT '',
    price               bigint NOT NULL CHECK (price >= 0),
    duration_days       int NOT NULL DEFAULT 30 CHECK (duration_days > 0),
    grace_days          int NOT NULL DEFAULT 7 CHECK (grace_days >= 0),
    max_invitations     int NOT NULL DEFAULT 1 CHECK (max_invitations > 0),
    max_guests          int NOT NULL DEFAULT 100 CHECK (max_guests > 0),
    allow_custom_domain boolean NOT NULL DEFAULT false,
    is_active           boolean NOT NULL DEFAULT true,
    sort_order          int NOT NULL DEFAULT 0,
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- Batas paket di-snapshot ke langganan supaya perubahan harga/kuota paket tidak
-- mengubah langganan yang sedang berjalan.
CREATE TABLE subscriptions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users (id),
    plan_id         uuid NOT NULL REFERENCES plans (id),
    status          subscription_status NOT NULL DEFAULT 'active',
    starts_at       timestamptz NOT NULL DEFAULT now(),
    ends_at         timestamptz NOT NULL,
    grace_ends_at   timestamptz NOT NULL,
    max_invitations int NOT NULL,
    max_guests      int NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX subscriptions_user_idx ON subscriptions (user_id, status);
CREATE INDEX subscriptions_lifecycle_idx ON subscriptions (status, ends_at, grace_ends_at);

CREATE TABLE orders (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code              text NOT NULL UNIQUE,
    user_id           uuid NOT NULL REFERENCES users (id),
    plan_id           uuid NOT NULL REFERENCES plans (id),
    price             bigint NOT NULL,
    unique_code       int NOT NULL,
    amount            bigint NOT NULL,
    status            order_status NOT NULL DEFAULT 'awaiting_payment',
    proof_path        text,                    -- relatif ke PRIVATE_DIR, tidak publik
    proof_uploaded_at timestamptz,
    reject_reason     text NOT NULL DEFAULT '',
    reviewed_by       uuid REFERENCES users (id),
    reviewed_at       timestamptz,
    subscription_id   uuid REFERENCES subscriptions (id),
    expires_at        timestamptz NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX orders_status_idx ON orders (status, created_at DESC);
CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);

-- Katalog tema; disinkronkan dari folder themes/ saat startup.
CREATE TABLE themes (
    slug        text PRIMARY KEY,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    tags        text[] NOT NULL DEFAULT '{}',
    colors      text[] NOT NULL DEFAULT '{}',
    fonts       text[] NOT NULL DEFAULT '{}',
    is_active   boolean NOT NULL DEFAULT true,
    is_premium  boolean NOT NULL DEFAULT false,
    sort_order  int NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE invitations (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users (id),
    theme           text REFERENCES themes (slug),
    theme_locked_at timestamptz,               -- customer hanya boleh memilih tema sekali
    status          invitation_status NOT NULL DEFAULT 'draft',
    content         jsonb NOT NULL DEFAULT '{}',
    published_at    timestamptz,
    suspended_at    timestamptz,               -- diisi lifecycle saat langganan expired (penanda untuk restore)
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);
CREATE INDEX invitations_user_idx ON invitations (user_id) WHERE deleted_at IS NULL;

CREATE TABLE domains (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id uuid NOT NULL REFERENCES invitations (id),
    kind          domain_kind NOT NULL,
    hostname      text NOT NULL,
    verify_token  text,
    verified_at   timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
-- Unik hanya untuk baris aktif: setelah soft delete, nama bisa dipakai orang lain.
CREATE UNIQUE INDEX domains_hostname_uq ON domains (lower(hostname)) WHERE deleted_at IS NULL;
CREATE INDEX domains_invitation_idx ON domains (invitation_id);

CREATE TABLE guests (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id uuid NOT NULL REFERENCES invitations (id) ON DELETE CASCADE,
    name          text NOT NULL,
    phone         text NOT NULL DEFAULT '',
    group_name    text NOT NULL DEFAULT '',
    pax           int NOT NULL DEFAULT 1 CHECK (pax BETWEEN 1 AND 20),
    code          text NOT NULL,
    opened_at     timestamptz,
    checked_in_at timestamptz,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE UNIQUE INDEX guests_code_uq ON guests (invitation_id, code);
CREATE INDEX guests_invitation_idx ON guests (invitation_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE wishes (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    invitation_id uuid NOT NULL REFERENCES invitations (id) ON DELETE CASCADE,
    guest_id      uuid REFERENCES guests (id) ON DELETE SET NULL,
    name          text NOT NULL,
    attendance    attendance NOT NULL,
    pax           int NOT NULL DEFAULT 1,
    message       text NOT NULL DEFAULT '',
    ip            text NOT NULL DEFAULT '',
    is_hidden     boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);
CREATE INDEX wishes_invitation_idx ON wishes (invitation_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE uploads (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid REFERENCES users (id),
    kind       text NOT NULL,
    path       text NOT NULL,
    size_bytes bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE music_tracks (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title            text NOT NULL,
    artist           text NOT NULL DEFAULT '',
    url              text NOT NULL,
    duration_seconds int NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);

CREATE TABLE settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Log pengingat/lifecycle supaya cron idempotent.
CREATE TABLE notifications (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL REFERENCES users (id),
    subscription_id uuid REFERENCES subscriptions (id),
    kind            text NOT NULL,
    channel         text NOT NULL DEFAULT 'in_app',
    created_at      timestamptz NOT NULL DEFAULT now(),
    read_at         timestamptz
);
CREATE UNIQUE INDEX notifications_once_uq ON notifications (subscription_id, kind, channel);

CREATE TABLE audit_logs (
    id         bigserial PRIMARY KEY,
    actor_id   uuid REFERENCES users (id),
    action     text NOT NULL,
    target     text NOT NULL DEFAULT '',
    meta       jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_logs_created_idx ON audit_logs (created_at DESC);
