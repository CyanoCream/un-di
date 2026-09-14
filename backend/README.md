# Undangan Digital — Backend

Go workspace (DDD, siap dipecah microservice) yang melayani **semua host dalam satu binary**:

| Host | Isi |
|---|---|
| `undangin.id`, `www.` | landing page + katalog tema (`/tema`) |
| `admin.undangin.id` | portal super admin (build dari repo frontend) |
| `app.undangin.id` | portal customer + stasiun check-in (build dari repo frontend) |
| `<slug>.undangin.id` / domain customer | halaman undangan (20 tema) |
| `/api/v1/*` (host mana pun) | JSON API |

Dokumentasi: [`docs/BACKEND.md`](docs/BACKEND.md) (arsitektur & port antar-service) · [`docs/SPEC.md`](docs/SPEC.md) (kontrak data & API) ·
[`docs/THEMES.md`](docs/THEMES.md) (katalog & aturan tema) · [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) (keputusan awal).

## Kebutuhan
Go 1.26+, PostgreSQL 15+ (dikembangkan dengan 17). Opsional: Google Chrome (thumbnail tema), Docker.

## Menjalankan (dev)

```bash
cp .env.example .env            # isi DATABASE_URL, JWT_SECRET, SEED_ADMIN_EMAIL/PASSWORD
go run ./app/cmd/server         # migrasi otomatis, sinkron tema, buat super admin & paket default → :8080
```

- Landing: http://localhost:8080 · Undangan: `http://<subdomain>.localhost:8080/<nama-tamu>` · Preview tema: http://localhost:8080/_preview/jawa-sogan
- Portal: jalankan repo frontend (`npm run dev:portal` :5173, `npm run dev:admin` :5174) — Vite mem-proxy API ke :8080.
- Pengembangan tema tanpa DB: `go run ./app/cmd/themedev` → http://localhost:8090 · thumbnail: `node tools/themeshot.mjs thumb <slug>`
- Landing tanpa DB: `go run ./app/cmd/landingdev` → http://localhost:8095

## Test

```bash
go test ./kernel/... ./app/... ./services/auth/... ./services/user/... ./services/billing/... \
        ./services/invitation/... ./services/theme/... ./services/media/... ./services/dashboard/... \
        ./services/landing/... ./services/audit/...
```
Integration test DB berjalan bila env berikut diisi: `BILLING_TEST_DATABASE_URL`, `MEDIA_TEST_DATABASE_URL`, `S3_TEST_ENDPOINT` (+ key).

## Database: migrasi, backup, pindah server

| Kebutuhan | Perintah |
|---|---|
| Migrasi (tanpa Go, pakai psql) | `./scripts/db-migrate.sh` · status: `./scripts/db-migrate.sh --status` |
| Migrasi (Go, lintas OS) | `go run ./app/cmd/migrate up` · `status` · `print` (cetak semua SQL) |
| Backup DB + file storage | `./scripts/db-backup.sh` → `backups/undangan-<waktu>/` |
| Restore di server baru | `./scripts/db-restore.sh backups/undangan-<waktu>` |

File SQL ada di [`migrations/`](migrations/) (berurutan, satu transaksi per file, dicatat di `schema_migrations`).
Server juga menjalankan migrasi otomatis saat start — ketiga cara aman dipakai bergantian.

Panduan lengkap VPS: [`docs/DEPLOY.md`](docs/DEPLOY.md).

**Pindah server:** `db-backup.sh` di server lama → salin folder backup + `.env` + folder `dist` frontend →
di server baru `docker compose up -d db` → `db-restore.sh <folder>` → `docker compose up -d`.
Bila `STORAGE_DRIVER=s3`, file sudah di object storage sehingga hanya database yang dipindah.

## Deploy

```bash
# repo frontend: npm run build → salin apps/admin/dist & apps/portal/dist ke backend/deploy/dist/{admin,portal}
cd deploy && docker compose up -d --build     # PostgreSQL + backend + Caddy
```
Tanpa Docker: `go build -o server ./app/cmd/server`, jalankan dari folder yang berisi `themes/`, `web/`, `.env`,
lalu Caddy memakai `deploy/Caddyfile`. DNS: `undangin.id` dan `*.undangin.id` → IP server.
