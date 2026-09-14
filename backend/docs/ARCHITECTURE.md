# Arsitektur Undangan Digital (draft v0)

## 1. Tiga aplikasi, satu backend

| App | Domain | Stack | Siapa |
|---|---|---|---|
| **Undangan (renderer)** | `<slug>.undangin.id` / domain customer | Go `html/template` + CSS + vanilla JS | Tamu undangan |
| **Portal customer** | `app.undangin.id` | Vue 3 + Vite SPA (`apps/portal`) | Pasangan / pemesan |
| **Portal admin** | `admin.undangin.id` | Vue 3 + Vite SPA (`apps/admin`) | Super admin |
| **API** | `api.undangin.id` | Go workspace (repo `backend`) | Dipakai kedua SPA |
| **Landing page** | `undangin.id` | Go template (atau static) | Calon customer |

`undangin.id` = placeholder, ganti dengan brand asli (`BASE_DOMAIN`).

### Kenapa bukan Nuxt untuk undangan?
Halaman undangan dibuka ratusan tamu dari HP, sering sinyal jelek, dan **link dishare via WhatsApp** → butuh
SSR untuk OG meta. Go render HTML langsung: tanpa runtime Node, tanpa hydration, payload JS ~1–2 KB.
Satu binary Go sanggup melayani ribuan request/detik di VPS 1–2 GB.

Kalau nanti tema butuh interaksi berat (galeri swipe, animasi scroll), cukup tambah Alpine.js / GSAP
**per tema**, bukan framework global. Alternatif jika tim desainer lebih nyaman komponen: Astro SSR + Vue islands.

### Kenapa SPA Vite, bukan Nuxt, untuk portal?
Portal di balik login → tidak perlu SEO/SSR. SPA = file statis dilayani Caddy, nol biaya server,
build cepat. Admin dipisah dari portal supaya kode admin tidak ikut ter-bundle ke customer
dan bisa dipasang proteksi tambahan (IP allowlist / Cloudflare Access).

### Go vs Supabase
| | Go + Postgres (self-host VPS) | Supabase |
|---|---|---|
| Biaya awal | VPS ±Rp100–200rb/bln, flat | Free tier: DB 500 MB, storage 1 GB, project di-pause jika idle; Pro $25/bln |
| Multi-tenant host routing, render tema | Natural | Tetap butuh server terpisah (Supabase tidak render HTML per host) |
| Cron lifecycle langganan | goroutine / cron di binary yang sama | pg_cron + Edge Functions |
| Caddy on-demand TLS `ask` | Endpoint internal biasa | Butuh server lain juga |
| Webhook payment | Handler biasa | Edge Function |
| Lock-in | Tidak | Sedang |

Keputusan: **Go + PostgreSQL**. Karena renderer undangan + TLS ask + cron wajib ada server sendiri,
Supabase hanya menambah satu dependensi lagi. Storage foto: **Cloudflare R2** (S3-compatible, tanpa biaya egress —
penting karena undangan penuh foto).

## 2. Subdomain & custom domain

```
Tamu ──> Cloudflare DNS ──> VPS: Caddy ──┬─ admin.* / app.*  → file statis SPA
                                         └─ lainnya          → Go :8080 (routing by Host)
Caddy on-demand TLS ──ask──> Go 127.0.0.1:8081/internal/tls/ask
```

- **Subdomain**: DNS wildcard `*.undangin.id → VPS`. Satu **wildcard certificate** via DNS challenge Cloudflare.
  Customer pilih slug di portal → `GET /v1/subdomains/check?name=` → langsung aktif, tidak ada provisioning DNS per user.
  **Jangan** on-demand TLS untuk subdomain: Let's Encrypt limit 50 cert/domain/minggu.
- **Custom domain** (customer beli sendiri):
  1. Customer input `budiani.com` di portal.
  2. Instruksi DNS: `CNAME www → custom.undangin.id` (atau A record apex → IP VPS) + `TXT _undangan-verify` berisi token.
  3. Tombol "Verifikasi" → API cek DNS → `domains.verified_at` terisi.
  4. Request pertama ke domain → Caddy tanya `ask` → Go jawab 200 → cert Let's Encrypt terbit otomatis.
- **Jual domain dari portal** (fase 2): API reseller registrar. Untuk `.id`/`.my.id`/`.web.id` harus registrar
  terakreditasi PANDI. Karena DNS diarahkan otomatis saat pembelian, verifikasi bisa dilewati.
- Opsi skala besar: Cloudflare for SaaS (custom hostnames) menggantikan Caddy on-demand.

## 3. Paket & lifecycle langganan

Contoh paket: Rp100.000 → 30 hari, 1 undangan, maks 100 tamu, subdomain gratis. Semua angka ada di tabel `plans`
(diubah dari admin, bukan hardcode).

```
pending ──bayar (webhook)──> active ──ends_at──> grace (7 hari) ──grace_ends_at──> expired
                               ^                    │                                │
                               └──── perpanjang ────┘         reaktivasi (bayar lagi)┘
```

Cron (tiap jam, idempotent via tabel `notifications`):

| Waktu | Aksi |
|---|---|
| H-7, H-3, H-1 sebelum `ends_at` | Kirim pengingat email + WhatsApp, link perpanjang |
| `ends_at` lewat | Status `grace`. Portal: edit dikunci, banner "perpanjang". **Undangan tetap online** (bisa jadi acara belum lewat). |
| `grace_ends_at` lewat | Status `expired`. Undangan `suspended` → 404/halaman "tidak aktif". `domains.deleted_at = now()` → subdomain lepas & bisa diambil orang. Undangan, tamu, RSVP di-soft-delete. |
| expired + 90 hari | Hapus foto di R2 (hemat storage). Data teks tetap soft-deleted. |
| expired + 1 tahun | Hard delete data pribadi tamu (nama, no HP) — kepatuhan **UU PDP**. Tulis di Syarat & Ketentuan. |

**Reaktivasi**: bayar → restore `deleted_at = NULL`. Subdomain lama dipulihkan jika belum dipakai orang lain
(unique index parsial `WHERE deleted_at IS NULL` memungkinkan ini); jika sudah dipakai, customer diminta pilih slug baru.

## 4. Tema

- Satu tema = satu folder `backend/themes/<slug>/` berisi `index.html` + `assets/`.
- Data yang tersedia di template: `.C.Groom`, `.C.Bride`, `.C.Events`, `.C.Gallery`, `.Guest` (dari `?to=`), `.AssetBase`.
- Konversi template referensi (HTML) → ganti teks statis dengan `{{.C.Groom.Nickname}}` dsb.
- Preview di portal: iframe ke renderer dengan data draft.
- Aset tema di-cache `immutable` 1 tahun; halaman undangan `private, max-age=60`.

## 5. Payment gateway
Midtrans / Xendit / Tripay (QRIS, VA, e-wallet). Webhook → verifikasi signature → idempotent via
`payments (provider, provider_ref)` unique → aktifkan/perpanjang subscription dalam satu transaksi DB.

## 6. Roadmap
1. **MVP**: auth, CRUD undangan + 2–3 tema, subdomain, tamu + link personal, RSVP/ucapan, admin kelola user/paket/tema, payment + lifecycle cron.
2. Custom domain (verifikasi DNS + on-demand TLS), kirim undangan via WhatsApp, statistik dibuka.
3. Jual domain via registrar API, tema premium, amplop digital / QRIS hadiah.

## Pertanyaan terbuka
- "100rb untuk 100 user" = 100 **tamu** per undangan, atau 100 **undangan**? (asumsi sekarang: 100 tamu, 1 undangan)
- Masa aktif 30 hari dihitung dari bayar atau dari publish?
- Selama grace, undangan tetap online? (asumsi: ya)
