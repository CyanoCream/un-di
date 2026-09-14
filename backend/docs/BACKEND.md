# Arsitektur Backend (Go, DDD)

Repo backend = **Go workspace** (`go.work`). Setiap service adalah **modul Go terpisah** yang hanya bergantung ke `kernel`,
sehingga siap dipecah menjadi microservice tanpa menulis ulang kode.

```
backend/
  go.work                   daftar modul (monolith: semua dijalankan satu binary)
  kernel/                   modul bersama, TIDAK tahu domain bisnis
    apperror/               error domain → status HTTP
    database/               pgx pool, migrator (Migrate/Status), TxManager + Conn(ctx)
    httpx/                  JSON, Fail, Decode, paginasi, Router (Public/Auth/Admin/Customer), middleware
    storage/                FileStorage (local | S3-compatible)
    security/               PasswordHasher (bcrypt), token/kode acak
    authctx/                Principal user login di context
    tenant/                 Host → platform / admin / app / subdomain / custom domain
    audit/                  port Recorder (diimplementasikan service audit)
    normalize/              normalisasi email & no HP
  migrations/               file SQL berurutan + embed (dipakai server, cmd/migrate, scripts/db-migrate.sh)
  services/
    <service>/              go.mod sendiri
      domain/               entity, aturan bisnis, INTERFACE repository & port ke service lain
      repository/           implementasi Postgres
      service/              interface Service + implementasi use case
      controller/           HTTP handler; Register(rt *httpx.Router)
  app/                      composition root (modul app)
    cmd/server/             monolith: wiring semua service, HTTP publik + internal
    cmd/migrate/            up | status | print
    cmd/themedev/           pengembangan tema tanpa DB
    cmd/landingdev/         pengembangan landing page tanpa DB
    internal/config/        env + .env
    internal/server/        router by Host + SPA admin/portal
    internal/adapters/      penghubung antar-service in-process (Accounts, ThemeCatalog, Landing*)
  themes/                   tema undangan (html/template) + _shared runtime
  web/landing/              template landing page
  scripts/                  db-migrate.sh, db-backup.sh, db-restore.sh
  deploy/                   Dockerfile, docker-compose.yml, Caddyfile
  tools/                    themeshot.mjs (thumbnail & screenshot tema)
```

## Memecah jadi microservice (nanti)

1. Buat `services/<nama>/cmd/<nama>/main.go` yang merakit service itu saja (repository → service → controller → httpx.Router).
2. Di `app/internal/adapters`, ganti adapter in-process untuk service tersebut dengan client HTTP/gRPC yang memenuhi port yang sama.
3. Pindahkan tabel milik service (lihat komentar di `migrations/migrations.go`) ke database sendiri; hapus foreign key lintas service.
4. Token JWT sudah stateless → service lain cukup memverifikasi dengan secret/kunci publik yang sama.

Ketergantungan antar-service saat ini **hanya lewat interface**:
| Pemakai → penyedia | Port | Adapter |
|---|---|---|
| auth → user | `auth/domain.AccountStore` | `adapters.Accounts` |
| user → auth | `user/domain.SessionRevoker` | repository refresh token |
| user (admin detail) → billing, invitation | `SubscriptionFinder`, `SubscriptionGranter`, `InvitationLister` | service billing, controller invitation |
| invitation → billing | `SubscriptionReader` | `billing/service.InvitationQuotaAdapter` |
| invitation → theme | `ThemeChecker` | service theme |
| billing → invitation | `InvitationLifecycle` | service invitation |
| theme → invitation (renderer) | `theme/domain.Catalog` | `adapters.ThemeCatalog` |
| landing → theme, billing | `ThemeLister`, `PlanLister`, `ContactProvider` | `adapters.Landing*` |
| semua → audit | `kernel/audit.Recorder` | service audit |

## Modul

| Modul | Isi |
|---|---|
| `auth` | Login/register/refresh/logout, JWT (`auth/jwt`), refresh token rotasi, middleware |
| `user` | Profil, manajemen user oleh admin, reset password, suspend |
| `audit` | `Recorder` port + daftar audit log |
| `billing` | Paket, order pembayaran manual QRIS, langganan, pengaturan pembayaran, job lifecycle |
| `theme` | Katalog tema (sinkron folder, kategori, thumbnail) |
| `invitation` | Undangan, subdomain & domain sendiri, tamu + import, ucapan/RSVP, check-in, konfirmasi hadiah, renderer tema & halaman publik |
| `media` | Upload file, pustaka musik |
| `dashboard` | Statistik admin (read model lintas tabel) |
| `landing` | Landing page marketing + katalog tema |

## Aturan

1. **Arah dependensi:** `controller → service → domain ← repository`. Domain tidak mengimpor pgx/http.
2. **Lintas modul lewat interface (port)** yang didefinisikan di modul PEMAKAI. Contoh: `invitation/domain.SubscriptionReader`
   diimplementasikan oleh `billing`. Wiring dilakukan di `app/cmd/server` (+ `app/internal/adapters`).
3. **Transaksi:** service memanggil `tx.WithinTx(ctx, func(ctx) error {...})`; repository selalu memakai `database.Conn(ctx, pool)`.
4. **Error:** repository mengembalikan `apperror.ErrNotFound` untuk baris tidak ada; service mengembalikan `apperror.*`
   dengan pesan Bahasa Indonesia; controller cukup `return err`.
5. **Otorisasi:** Router menjaga role (`rt.Admin`, `rt.Customer`, `rt.Auth`). Kepemilikan data dicek di **service**
   (customer hanya data miliknya → balas 404 bila bukan miliknya).
6. **Audit:** aksi admin yang mengubah data customer → `audit.Record(...)`.

## Auth (JWT)

- Access token: JWT HS256, 15 menit, klaim `sub, role, name, email, portal, iss, aud, exp, jti`.
  Dikirim sebagai cookie `undangan_at` (HttpOnly, SameSite=Lax) **atau** header `Authorization: Bearer` (klien mobile).
- Refresh token: acak 256-bit, disimpan sebagai SHA-256 di `refresh_tokens`, cookie `undangan_rt`
  (HttpOnly, SameSite=Strict, path `/api/v1/auth`), 30 hari. **Dirotasi** tiap `/auth/refresh`;
  token yang sudah dicabut dipakai lagi → seluruh family dicabut.
- Access token kedaluwarsa → 401 `token_expired` → frontend otomatis memanggil `/auth/refresh` lalu mengulang request.
- Suspend user / reset password → semua refresh token dicabut (access token yang tersisa habis ≤ 15 menit).
- Request non-GET wajib header `X-Requested-With` (CSRF) kecuali memakai Bearer.
