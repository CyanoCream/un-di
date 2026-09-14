// Package migrations = skema PostgreSQL berupa file SQL berurutan.
//
// File yang sama dipakai tiga cara:
//   - otomatis saat server start (cmd/server),
//   - `go run ./app/cmd/migrate up|status|print`,
//   - manual: `scripts/db-migrate.sh` (psql).
//
// Semua mencatat versi di tabel schema_migrations (version = nama file tanpa .sql), jadi aman dicampur.
//
// Kepemilikan tabel per service (untuk dipecah ke database terpisah nanti):
//
//	user: users · auth: refresh_tokens · audit: audit_logs
//	billing: plans, subscriptions, orders, settings, notifications
//	theme: themes · media: uploads, music_tracks
//	invitation: invitations, domains, guests, wishes, checkin_logs, gift_confirmations
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
