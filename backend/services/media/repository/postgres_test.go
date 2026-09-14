package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/services/media/domain"
	"undangan/kernel/apperror"
	"undangan/kernel/database"
)

// Integrasi: jalankan dengan MEDIA_TEST_DATABASE_URL yang sudah dimigrasi (0001_init.sql).
func TestPostgresIntegration(t *testing.T) {
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

	// Semua di dalam transaksi yang di-rollback → DB tetap bersih & tes bisa diulang.
	errRollback := errors.New("rollback")
	err = database.NewTxManager(pool).WithinTx(ctx, func(ctx context.Context) error {
		runMediaQueries(ctx, t, pool)
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
}

func runMediaQueries(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	// uploads: tanpa user & dengan user
	uploads := NewUploads(pool)
	anon := &domain.Upload{Kind: domain.KindImage, Path: "2026/09/a.png", SizeBytes: 123}
	if err := uploads.Create(ctx, anon); err != nil || anon.ID == "" || anon.CreatedAt.IsZero() {
		t.Fatalf("create anon upload: %v %+v", err, anon)
	}
	var userID string
	if err := database.Conn(ctx, pool).QueryRow(ctx, `INSERT INTO users (name, email, password_hash) VALUES ('Uji', 'media-'||gen_random_uuid()||'@x.test', 'x') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	withUser := &domain.Upload{UserID: &userID, Kind: domain.KindAudio, Path: "2026/09/b.mp3", SizeBytes: 456}
	if err := uploads.Create(ctx, withUser); err != nil {
		t.Fatalf("create user upload: %v", err)
	}

	// music: insert / list / soft delete
	music := NewMusic(pool)
	before, err := music.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tr := &domain.MusicTrack{Title: "Canon in D", Artist: "Pachelbel", URL: "/uploads/2026/09/b.mp3"}
	if err := music.Create(ctx, tr); err != nil || tr.ID == "" {
		t.Fatalf("create music: %v %+v", err, tr)
	}
	after, err := music.List(ctx)
	if err != nil || len(after) != len(before)+1 {
		t.Fatalf("list after create: %v len=%d before=%d", err, len(after), len(before))
	}
	if got := after[0]; got.ID != tr.ID || got.Title != tr.Title || got.Artist != tr.Artist || got.URL != tr.URL {
		t.Fatalf("unexpected first track %+v", got)
	}
	if err := music.SoftDelete(ctx, tr.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if err := music.SoftDelete(ctx, tr.ID); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("second delete want ErrNotFound, got %v", err)
	}
	if err := music.SoftDelete(ctx, "bukan-uuid"); !errors.Is(err, apperror.ErrNotFound) {
		t.Fatalf("invalid uuid want ErrNotFound, got %v", err)
	}
	final, err := music.List(ctx)
	if err != nil || len(final) != len(before) {
		t.Fatalf("list after delete: %v len=%d", err, len(final))
	}
}
