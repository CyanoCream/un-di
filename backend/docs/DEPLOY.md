# Deploy ke VPS

Satu VPS menjalankan semuanya dengan Docker: **PostgreSQL + backend Go + Caddy (SSL otomatis)**.
Contoh memakai domain `undangin.id` — ganti dengan domain kamu.

```
Internet ──► Caddy :443 ──► app :8080 ──► PostgreSQL
               │               ├─ undangin.id            landing page
               │               ├─ app.undangin.id        portal customer
               │               ├─ admin.undangin.id      portal admin
               │               ├─ <slug>.undangin.id     undangan
               │               └─ domain customer        undangan
               └─ SSL: wildcard *.undangin.id (Cloudflare DNS) + otomatis untuk domain customer
```

## 0. Yang disiapkan

| Kebutuhan | Rekomendasi |
|---|---|
| VPS | Ubuntu 24.04, **2 vCPU / 2 GB RAM / 40 GB SSD** (cukup untuk ribuan undangan). Lokasi Indonesia/Singapura |
| Domain | mis. `undangin.id` |
| Cloudflare (gratis) | Nameserver domain dipindah ke Cloudflare — dibutuhkan untuk SSL wildcard |
| Git | Repo `github.com/CyanoCream/un-di` (berisi folder `backend/` & `frontend/`) |

## 1. DNS di Cloudflare

| Type | Name | Content | Proxy |
|---|---|---|---|
| A | `@` | IP VPS | **DNS only** (awan abu-abu) |
| A | `*` | IP VPS | **DNS only** |
| A | `custom` | IP VPS | **DNS only** — target CNAME untuk domain customer |

Buat **API Token** Cloudflare: *My Profile → API Tokens → Create Token → template "Edit zone DNS"*, zona `undangin.id`.
Simpan untuk `CF_API_TOKEN`.

> Proxy (awan oranye) dimatikan supaya Caddy yang memegang SSL dan domain customer bisa diverifikasi ke IP VPS.

## 2. Siapkan server (sekali saja)

```bash
ssh root@IP_VPS

# user non-root + firewall
adduser deploy && usermod -aG sudo deploy
ufw allow OpenSSH && ufw allow 80 && ufw allow 443 && ufw --force enable

# Docker + Compose plugin
curl -fsSL https://get.docker.com | sh
usermod -aG docker deploy

# swap 2 GB (aman untuk build di VPS RAM kecil)
fallocate -l 2G /swapfile && chmod 600 /swapfile && mkswap /swapfile && swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
timedatectl set-timezone Asia/Jakarta
```

Login ulang sebagai `deploy`.

## 3. Ambil kode

```bash
git clone https://github.com/CyanoCream/un-di.git ~/undangan
# repo privat: pakai Personal Access Token GitHub sebagai password, atau deploy key SSH
```

## 4. Isi konfigurasi `backend/.env`

```bash
cd ~/undangan/backend
cp .env.example .env
nano .env
```

Nilai penting untuk production:

```dotenv
APP_ENV=production
APP_NAME=Undangin
BASE_DOMAIN=undangin.id
PUBLIC_URL_PATTERN=https://{sub}.undangin.id
PORTAL_URL=https://app.undangin.id
CUSTOM_DOMAIN_CNAME=custom.undangin.id
CUSTOM_DOMAIN_IPS=IP_VPS

POSTGRES_PASSWORD=GANTI_PASSWORD_KUAT
DATABASE_URL=postgres://undangan:GANTI_PASSWORD_KUAT@db:5432/undangan?sslmode=disable

JWT_SECRET=HASIL_openssl_rand_-base64_48
SEED_ADMIN_NAME=Super Admin
SEED_ADMIN_EMAIL=admin@undangin.id
SEED_ADMIN_PASSWORD=PASSWORD_ADMIN_KUAT

ACME_EMAIL=admin@undangin.id
CF_API_TOKEN=TOKEN_CLOUDFLARE

STORAGE_DRIVER=local          # atau s3 + S3_* bila object storage sudah siap
```

Buat secret: `openssl rand -base64 48`. File `.env` jangan pernah di-commit.

## 5. Build frontend & jalankan

```bash
cd ~/undangan/backend
mkdir -p deploy/backups && sudo chown 10001:10001 deploy/backups   # user di dalam container
./deploy/build-frontend.sh          # build Vue pakai container Node → deploy/dist
cd deploy
docker compose up -d --build        # build pertama ± 3–5 menit
docker compose ps
docker compose logs -f app          # tunggu "listening" ; Ctrl+C untuk keluar
```

Saat start, backend otomatis: menjalankan migrasi SQL, sinkron 20 tema, membuat paket default, dan membuat super admin dari `SEED_ADMIN_*`.

## 6. Cek

| URL | Harapan |
|---|---|
| https://undangin.id | Landing page |
| https://admin.undangin.id | Login super admin |
| https://app.undangin.id | Daftar / masuk customer |
| https://undangin.id/_preview/jawa-sogan | Demo tema |

Setelah login admin: **Pengaturan → Pembayaran** (upload QRIS, nomor WA admin), cek **Paket** (harga & fitur).
Setelah itu kosongkan `SEED_ADMIN_PASSWORD` di `.env` (admin sudah dibuat).

## 7. Backup otomatis harian

```bash
crontab -e
# setiap jam 02.00: backup + hapus backup lebih dari 14 hari
0 2 * * * cd /home/deploy/undangan/backend/deploy && docker compose exec -T app ./scripts/db-backup.sh >> backups/cron.log 2>&1 && find backups -maxdepth 1 -name 'undangan-*' -mtime +14 -exec rm -rf {} +
```

Salin folder `deploy/backups` ke tempat lain secara berkala (object storage / komputer lain) — backup di server yang sama tidak melindungi dari server rusak.

## 8. Update aplikasi

```bash
cd ~/undangan/backend && ./deploy/update.sh
```
Script: `git pull` → backup database → build frontend → build & restart backend (migrasi otomatis).

## 9. Pindah server

Di server lama:
```bash
cd ~/undangan/backend/deploy
docker compose exec -T app ./scripts/db-backup.sh
scp -r backups/undangan-<waktu> ../.env deploy@IP_BARU:~/
```

Di server baru (langkah 2–3, taruh `.env` di `~/undangan/backend/.env`, arahkan DNS ke IP baru):
```bash
cd ~/undangan/backend
mkdir -p deploy/backups && sudo chown 10001:10001 deploy/backups
mv ~/undangan-<waktu> deploy/backups/
./deploy/build-frontend.sh
cd deploy
docker compose up -d --build db app
docker compose exec app ./scripts/db-restore.sh backups/undangan-<waktu>   # ketik "ya"
docker compose up -d caddy
```

## Perintah harian

| Perintah (di `backend/deploy`) | Fungsi |
|---|---|
| `docker compose ps` | status container |
| `docker compose logs -f app` | log aplikasi |
| `docker compose restart app` | restart backend |
| `docker compose exec app ./migrate status` | status migrasi |
| `docker compose exec db psql -U undangan` | masuk database |
| `docker compose down` | matikan semua (data tetap aman di volume) |

## Masalah umum

| Gejala | Penyebab & solusi |
|---|---|
| SSL gagal / "certificate" di log caddy | `CF_API_TOKEN` salah/izin kurang; DNS belum mengarah; proxy Cloudflare masih oranye |
| `admin.` / `app.` tampil "belum di-build" | jalankan `./deploy/build-frontend.sh` lalu `docker compose restart app` |
| Backend restart terus | lihat `docker compose logs app`: biasanya `DATABASE_URL`/`JWT_SECRET` kosong |
| Domain customer tidak bisa diverifikasi | pastikan record `custom.undangin.id` ada & `CUSTOM_DOMAIN_IPS` berisi IP VPS |
