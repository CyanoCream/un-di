package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"undangan/kernel/database"
)

// Integrasi: jalankan dengan MEDIA_TEST_DATABASE_URL yang sudah dimigrasi (0001_init.sql).
func TestStatsIntegration(t *testing.T) {
	url := os.Getenv("MEDIA_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MEDIA_TEST_DATABASE_URL tidak diset")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := NewPostgres(pool)

	// Semua di dalam satu transaksi yang di-rollback → DB tetap bersih & tes bisa diulang.
	errRollback := errors.New("rollback")
	err = database.NewTxManager(pool).WithinTx(ctx, func(ctx context.Context) error {
		seedAndCheck(ctx, t, repo)
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
	if _, err := repo.Stats(context.Background()); err != nil {
		t.Fatalf("stats after rollback: %v", err)
	}
}

func seedAndCheck(ctx context.Context, t *testing.T, repo *Postgres) {
	t.Helper()
	conn := database.Conn(ctx, repo.pool)

	// ID acak supaya tidak bentrok dengan data yang sudah ada.
	var customer, deleted, plan, inv1, inv2 string
	if err := conn.QueryRow(ctx, `SELECT gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), gen_random_uuid()`).
		Scan(&customer, &deleted, &plan, &inv1, &inv2); err != nil {
		t.Fatal(err)
	}
	ids := strings.NewReplacer(
		"11111111-1111-1111-1111-111111111111", customer,
		"22222222-2222-2222-2222-222222222222", deleted,
		"33333333-3333-3333-3333-333333333333", plan,
		"44444444-4444-4444-4444-444444444444", inv1,
		"55555555-5555-5555-5555-555555555555", inv2,
		"'S-", "'S-"+customer[:8]+"-",
	)

	before, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("stats (before seed): %v", err)
	}

	seed := []string{
		`INSERT INTO users (id, name, email, password_hash) VALUES
			('11111111-1111-1111-1111-111111111111', 'Customer', 'stats-c-'||gen_random_uuid()||'@x.test', 'x'),
			('22222222-2222-2222-2222-222222222222', 'Hapus', 'stats-d-'||gen_random_uuid()||'@x.test', 'x')`,
		`UPDATE users SET deleted_at = now() WHERE id = '22222222-2222-2222-2222-222222222222'`,
		`INSERT INTO users (name, email, password_hash, role) VALUES ('Admin', 'stats-a-'||gen_random_uuid()||'@x.test', 'x', 'super_admin')`,
		`INSERT INTO plans (id, name, price) VALUES ('33333333-3333-3333-3333-333333333333', 'Basic', 50000)`,
		`INSERT INTO subscriptions (user_id, plan_id, status, ends_at, grace_ends_at, max_invitations, max_guests) VALUES
			('11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 'active', now() + interval '30 days', now() + interval '37 days', 1, 100),
			('11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 'grace', now(), now() + interval '7 days', 1, 100),
			('11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 'expired', now(), now(), 1, 100)`,
		`INSERT INTO orders (code, user_id, plan_id, price, unique_code, amount, status, reviewed_at, expires_at) VALUES
			('S-1', '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 50000, 123, 50123, 'paid', now(), now()),
			('S-2', '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 50000, 7, 50007, 'paid', date_trunc('month', now()) - interval '1 day', now()),
			('S-3', '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 50000, 9, 50009, 'awaiting_confirmation', NULL, now()),
			('S-4', '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 50000, 5, 50005, 'awaiting_payment', NULL, now())`,
		`INSERT INTO invitations (id, user_id, status) VALUES
			('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'published'),
			('55555555-5555-5555-5555-555555555555', '11111111-1111-1111-1111-111111111111', 'draft')`,
		`INSERT INTO invitations (user_id, status, deleted_at) VALUES ('11111111-1111-1111-1111-111111111111', 'published', now())`,
		`INSERT INTO guests (invitation_id, name, code, deleted_at) VALUES
			('44444444-4444-4444-4444-444444444444', 'Tamu 1', 'g1', NULL),
			('44444444-4444-4444-4444-444444444444', 'Tamu 2', 'g2', NULL),
			('44444444-4444-4444-4444-444444444444', 'Tamu 3', 'g3', now())`,
	}
	for _, q := range seed {
		if _, err := conn.Exec(ctx, ids.Replace(q)); err != nil {
			t.Fatalf("seed: %v\n%s", err, q)
		}
	}

	s, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("stats (after seed): %v", err)
	}
	checks := []struct {
		name      string
		got, want int64
	}{
		{"users", int64(s.Users - before.Users), 1},
		{"active_subscriptions", int64(s.ActiveSubscriptions - before.ActiveSubscriptions), 1},
		{"grace_subscriptions", int64(s.GraceSubscriptions - before.GraceSubscriptions), 1},
		{"pending_orders", int64(s.PendingOrders - before.PendingOrders), 1},
		{"invitations", int64(s.Invitations - before.Invitations), 2},
		{"published_invitations", int64(s.PublishedInvitations - before.PublishedInvitations), 1},
		{"revenue_this_month", s.RevenueThisMonth - before.RevenueThisMonth, 50123},
		{"guests", int64(s.Guests - before.Guests), 2},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s: delta %d, want %d", c.name, c.got, c.want)
		}
	}
}
