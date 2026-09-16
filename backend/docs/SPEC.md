# Spesifikasi v1 — kontrak antara API (Go), portal (Vue), dan tema

Dokumen ini sumber kebenaran. Backend, `packages/shared/src/types.ts`, dan tema harus konsisten dengan ini.

---

## 1. Hasil analisis referensi

| Referensi | Pola yang diambil |
|---|---|
| tamukita.invisimple.id | Cover "Kepada Yth. Bapak/Ibu/Saudara/i + nama + *Mohon maaf jika ada kesalahan penulisan nama/gelar*" → "Buka Undangan". Salam + QS Ar-Rum 21, mempelai (anak ke-, ortu, (Alm/Almh), Instagram), countdown + "Simpan Tanggal", acara per kartu, Love Story bertanggal, Wedding Gift (salin no. rekening), Wishes + RSVP (Hadir/Tidak Hadir + jumlah tamu 1–10), Terima Kasih + "Kami Yang Berbahagia", QR check-in tamu, musik loop. |
| ceritalendahardi.viding.co | Nuansa Jawa (joglo, gunungan), "Nyuwun Pangestu lan Donga Restu", gelar di nama, urutan **mempelai wanita dulu**, section judul bahasa Jawa, Google Calendar dengan timezone. |
| our-wedding.link | Menu navigasi bawah (Couple / Gallery / Date / Location / Wishes), zona WITA, acara adat "Ngunduh Mantu (Mapparola)", status "Acara telah selesai", ucapan dengan asal kota. |
| canva.site | Ngunduh mantu = jenis acara tersendiri (desain statis, tanpa fitur interaktif). |
| inv.asmaradigital.com | Countdown di atas, kutipan tokoh + ayat, 3 acara (Akad, Resepsi, Ngunduh Mantu), love story per periode, RSVP 3 opsi (Datang/Absen/Mungkin) + statistik, rekening + **alamat kirim kado + kontak**, "Kedua Mempelai & Keluarga Besar". |
| aadigital.my.id | **Link personal per tamu** (`/kode`), jumlah tamu per undangan, alamat singkat di cover, "Add to Calendar", galeri "Momen Pre-wedding", tiket QR (kode, tanggal, jam, nama, jumlah tamu). |

**Musik (semua referensi):** `<audio loop>` MP3, mulai **setelah klik "Buka Undangan"** (autoplay browser diblok tanpa interaksi), tombol bulat melayang kanan-bawah (ikon piringan berputar) untuk pause/play. Kita tambah: pause otomatis saat tab disembunyikan, `preload="none"` supaya tidak membebani load awal, dan titik mulai lagu (`start_at`).

> ⚠️ Referensi memakai lagu komersial (Brian McKnight, Reza Artamevia, Nadin Amizah). Pakai lagu berhak cipta di layanan berbayar berisiko klaim. Pustaka musik bawaan sebaiknya royalty-free / berlisensi; upload lagu oleh customer jadi tanggung jawab customer (tulis di S&K).

**Kelemahan referensi yang jadi keunggulan kita:** tamukita memuat 10 keluarga Google Font × 18 weight, WordPress + Elementor; halaman berat. Tema kita: 2 font, 1 CSS, 1 JS runtime (<10 KB), gambar lazy.

---

## 2. Data undangan (`invitations.content`, JSONB)

Semua key `snake_case`. Field kosong = section disembunyikan oleh tema.

```ts
interface InvitationContent {
  event_type: 'pernikahan' | 'ngunduh_mantu'
  religion: 'islam' | 'kristen' | 'katolik' | 'hindu' | 'buddha' | 'konghucu' | 'umum'
  couple_order: 'groom_first' | 'bride_first'

  cover: {
    title: string            // "The Wedding of"
    photo: string            // url
    background: string       // url opsional
  }
  opening: {
    greeting: string         // "Assalamu'alaikum Warahmatullahi Wabarakatuh" (default dari religion)
    text: string             // paragraf pembuka
  }
  quote: { text: string; source: string }   // default dari religion (QS Ar-Rum 21 / 1 Korintus 13:4-7 / ...)

  groom: Person
  bride: Person

  events: EventItem[]        // minimal 1 untuk publish

  love_story: { date: string; title: string; text: string; photo: string }[]

  gallery: {
    photos: string[]         // maks 20
    video_url: string        // YouTube
  }

  live_stream: { url: string; platform: string; note: string }  // url kosong = sembunyi

  gift: {
    enabled: boolean
    text: string
    accounts: { type: 'bank' | 'ewallet'; provider: string; number: string; holder: string }[]
    address: { recipient: string; phone: string; address: string }  // address kosong = sembunyi
  }

  rsvp: {
    enabled: boolean
    max_pax: number          // 1–10, pilihan jumlah tamu di form
    show_wishes: boolean     // tampilkan daftar ucapan
  }

  closing: {
    text: string
    sign_off: string         // "Wassalamu'alaikum ..." (default dari religion)
    from: string             // "Kami yang berbahagia, Kedua Mempelai & Keluarga Besar"
    family: string[]         // turut mengundang (opsional)
  }

  music: {
    enabled: boolean
    url: string              // dari pustaka musik atau upload customer (mp3)
    title: string
    start_at: number         // detik
  }

  share: {
    whatsapp_template: string  // placeholder {nama} {link}
  }
}

interface Person {
  full_name: string          // termasuk gelar: "Hardiansyah, S.Kom."
  nickname: string
  photo: string
  child_order: string        // "Putra pertama" / "Putri kedua" / "Putra bungsu"
  father: string             // "Bapak Suyadi"
  mother: string
  father_deceased: boolean   // tampil "(Alm.)"
  mother_deceased: boolean   // tampil "(Almh.)"
  instagram: string          // tanpa @
}

interface EventItem {
  id: string                 // uuid/nanoid client-side
  name: string               // "Akad Nikah" | "Resepsi" | "Pemberkatan" | "Ngunduh Mantu" | bebas
  date: string               // "2026-07-12"
  start_time: string         // "08:00"
  end_time: string           // "" → "Selesai"
  timezone: 'WIB' | 'WITA' | 'WIT'
  venue: string
  address: string
  maps_url: string
  note: string
}
```

Default per `religion` (diisi backend saat konten kosong): salam pembuka/penutup + kutipan.

---

## 3. Tamu (`guests`)

| Kolom | Keterangan |
|---|---|
| `name` | wajib, 1–100 char |
| `phone` | opsional, dinormalisasi `08xx` / `+62xx` → `628xx` |
| `group_name` | opsional ("Keluarga", "Teman Kantor") |
| `pax` | jumlah orang yang diundang, default 1 |
| `code` | 6 char `[A-Z0-9]` tanpa huruf ambigu, unik per undangan |
| `opened_at`, `checked_in_at` | pelacakan |

**Link tamu:**
- Personal: `https://<subdomain>.<base>/<code>` → nama & pax dari DB, `opened_at` tercatat.
- Generik: `https://<subdomain>.<base>/?to=Nama+Tamu` (juga terima slug `keluarga-dan-alumni` → "Keluarga Dan Alumni").

**Import CSV / XLSX** (`POST /invitations/{id}/guests/import`, multipart `file`):
- Baris pertama = header, case-insensitive. Alias:
  - nama: `nama`, `nama tamu`, `name`
  - no HP: `no_hp`, `no hp`, `hp`, `whatsapp`, `wa`, `phone`, `telepon`
  - grup: `grup`, `group`, `kategori`, `keterangan`
  - pax: `jumlah_tamu`, `jumlah tamu`, `pax`, `jumlah`
- Tanpa header dikenali → kolom 1 = nama, kolom 2 = no HP.
- `?dry_run=1` → validasi saja, balikan preview; tanpa dry_run → insert.
- Duplikat (nama lowercase + phone sama, sudah ada / dalam file) → dilewati.
- Kuota `max_guests` langganan dicek; baris yang melebihi sisa kuota ditandai `error` ("Melebihi kuota tamu paket"), baris `ok` tetap disimpan.
- Maks file 2 MB, 5.000 baris. Template: `GET /guests/template.xlsx` atau `template.csv`. CSV dengan pemisah `;` (Excel Indonesia) terdeteksi otomatis.

---

## 4. Role & permission

| Resource | super_admin | customer |
|---|---|---|
| Login | hanya di portal admin | hanya di portal customer |
| Users | list, buat, edit, suspend, reset password | profil & password sendiri |
| Paket (`plans`) | CRUD (nonaktifkan, tidak hapus) | lihat yang aktif |
| Tema | aktif/nonaktif, premium, urutan | lihat yang aktif + preview |
| Undangan | semua: buat untuk user mana pun, isi data, **ganti tema kapan saja**, subdomain, publish/unpublish, hapus (soft) | **milik sendiri saja**: buat (butuh langganan aktif & kuota), isi data, **pilih tema sekali** (terkunci setelah dipilih), subdomain selama belum publish, publish/unpublish |
| Tamu & import | semua undangan | undangan milik sendiri, dalam kuota |
| Ucapan/RSVP | lihat, sembunyikan, hapus semua | milik sendiri |
| Order pembayaran | lihat semua, **approve / reject** | buat, upload bukti, lihat milik sendiri |
| Langganan | lihat semua, beri/perpanjang manual | lihat milik sendiri |
| Pengaturan pembayaran (QRIS) | edit | lihat saat checkout |
| Pustaka musik | CRUD | pilih / upload sendiri |
| Audit log | lihat | — |

Aturan penegakan (backend):
- Scope customer selalu di SQL: `... AND user_id = $actor`. Resource milik orang lain → **404** (bukan 403, supaya tidak bocor keberadaan ID).
- Endpoint `/admin/*` → middleware `super_admin`.
- Tema terkunci: `invitations.theme_locked_at`. Customer `PUT theme` saat terkunci → 409. Admin memilihkan tema juga mengunci untuk customer.
- Aksi admin yang mengubah data customer → `audit_logs`.

---

## 5. Pembayaran manual (QRIS statis)

```
customer pilih paket → order dibuat: amount = harga + kode unik (1–499)
  status awaiting_payment ── upload bukti ──> awaiting_confirmation ── admin approve ──> paid → langganan aktif/diperpanjang
          │                                                  └── admin reject (alasan) ──> rejected (boleh upload ulang → awaiting_confirmation)
          └── 24 jam tanpa bukti ──> expired
```

- Admin upload gambar QRIS + nama merchant + instruksi + nomor WA admin di Pengaturan.
- Checkout menampilkan QRIS, **nominal persis** (dengan kode unik, supaya mudah dicocokkan di mutasi), tombol salin nominal, batas waktu, upload bukti (jpg/png/webp ≤ 5 MB), tombol "Konfirmasi via WhatsApp".
- Bukti bayar disimpan **privat** (bukan di folder publik); hanya pemilik order & admin yang bisa melihat.
- Approve: jika langganan `active`/`grace` → `ends_at = max(now, ends_at) + duration_days`; jika tidak → langganan baru mulai sekarang. Undangan yang ter-soft-delete karena expired dipulihkan.

---

## 6. API

Base: `/api/v1` di **host mana pun** (SPA dev lewat proxy Vite, prod lewat Caddy). JSON `snake_case`.

- Auth: JWT — lihat docs/BACKEND.md §Auth (cookie `undangan_at` + refresh `undangan_rt`, atau `Authorization: Bearer`).
- Semua request non-GET wajib header `X-Requested-With: fetch` (proteksi CSRF tambahan).
- Error: `{"error": {"code": "validation", "message": "...", "fields": {"email": "sudah terdaftar"}}}`.
- List: `{"items": [...], "total": 120, "page": 1, "per_page": 20}` dengan query `page`, `per_page`, `q`.

### Publik
| Method | Path | Ket |
|---|---|---|
| POST | `/auth/register` | `{name,email,phone,password}` → customer |
| POST | `/auth/login` | `{email,password,portal:"admin"\|"customer"}` |
| POST | `/auth/refresh` | rotasi refresh token (cookie) → access token baru |
| POST | `/auth/logout` | |
| GET | `/auth/me` | `{user}` / 401 |
| GET | `/subdomains/check?name=` | `{name,available,reason?}` |
| GET | `/public/invitations/{id}/wishes?page=` | `{items:[{name,attendance,pax,message,created_at}], stats:{hadir,tidak,ragu}, total}` |
| POST | `/public/invitations/{id}/wishes` | `{name,attendance,pax,message,guest_code?}` rate-limited |

### Login (dua role)
| Method | Path | Ket |
|---|---|---|
| GET | `/themes` | tema aktif |
| GET | `/plans` | paket aktif |
| GET | `/music` | pustaka musik |
| GET | `/payment-settings` | QRIS dsb |
| POST | `/uploads` | multipart `file`, `kind=image\|audio` → `{url}` |
| GET | `/invitations` | admin: semua (`?user_id=&q=`), customer: milik sendiri |
| POST | `/invitations` | `{user_id? (admin wajib), theme?, subdomain?, event_type}` |
| GET | `/invitations/{id}` | detail + `theme_locked`, `subdomain`, `url`, `status`, `guest_count`, `quota` |
| PATCH | `/invitations/{id}` | `{content}` |
| DELETE | `/invitations/{id}` | admin only |
| PUT | `/invitations/{id}/theme` | `{theme}` |
| PUT | `/invitations/{id}/subdomain` | `{name}` |
| POST | `/invitations/{id}/publish` / `/unpublish` | |
| GET | `/invitations/{id}/preview?theme=` | HTML (untuk iframe) |
| GET | `/invitations/{id}/guests?q=&group=&page=` | |
| POST | `/invitations/{id}/guests` | `{name,phone,group_name,pax}` |
| PATCH / DELETE | `/invitations/{id}/guests/{gid}` | |
| POST | `/invitations/{id}/guests/import?dry_run=1` | → `{rows:[{row,name,phone,group_name,pax,status:"ok"\|"duplicate"\|"error",error?}], summary:{ok,duplicate,error}, inserted}` |
| GET | `/invitations/{id}/guests/template.csv` · `template.xlsx` · `export.csv` · `export.xlsx` | |
| GET | `/invitations/{id}/wishes` | termasuk yang hidden |
| PATCH / DELETE | `/invitations/{id}/wishes/{wid}` | `{is_hidden}` |

### Customer (`/me`)
| Method | Path |
|---|---|
| GET / PATCH | `/me/profile` |
| POST | `/me/password` `{current_password,new_password}` |
| GET | `/me/subscription` → `{current, history}` |
| GET / POST | `/me/orders` (POST `{plan_id}`) |
| GET | `/me/orders/{id}` |
| POST | `/me/orders/{id}/proof` multipart `file` |
| GET | `/orders/{id}/proof` (pemilik & admin) → gambar |

### Admin (`/admin`)
| Method | Path |
|---|---|
| GET | `/admin/stats` |
| GET / POST | `/admin/users` |
| GET / PATCH | `/admin/users/{id}` (`{name,phone,is_suspended}`) |
| POST | `/admin/users/{id}/reset-password` → `{password}` sementara |
| POST | `/admin/users/{id}/subscriptions` `{plan_id, days?}` beri manual |
| GET / POST / PATCH | `/admin/plans`, `/admin/plans/{id}` |
| GET / PATCH | `/admin/themes`, `/admin/themes/{slug}` |
| GET | `/admin/orders?status=` , `/admin/orders/{id}` |
| POST | `/admin/orders/{id}/approve`, `/admin/orders/{id}/reject` `{reason}` |
| GET | `/admin/subscriptions?status=` |
| GET / PUT | `/admin/settings/payment` `{qris_image,merchant_name,instructions,admin_whatsapp}` |
| GET / POST / DELETE | `/admin/music`, `/admin/music/{id}` |
| GET | `/admin/audit-logs` |

---

## 7. Tema undangan

Struktur:
```
backend/themes/
  _shared/            partial template, runtime.js, base.css (di-serve di /_shared/)
  <slug>/
    theme.json        {"name","description","tags":[],"colors":["#..","#..","#.."],"fonts":["..",".."]}
    index.html        layout; memanggil partial dari _shared
    assets/           style.css, ornamen SVG
```

Katalog awal (desain original, tidak meniru referensi):

| Slug | Nama | Karakter |
|---|---|---|
| `jawa-sogan` | Jawa Sogan | Coklat sogan + emas, motif kawung/parang (SVG), gunungan, sapaan "Nyuwun Pangestu" |
| `islami-zamrud` | Islami Zamrud | Hijau zamrud + emas, pola bintang geometris, bingkai lengkung mihrab |
| `floral-sage` | Floral Sage | Rustic: sage + dusty pink, bunga line-art, kertas krem |
| `minimalis-monokrom` | Minimalis Monokrom | Off-white + hitam, tipografi besar, fokus foto |
| `royal-navy` | Royal Navy | Navy + emas, garis art-deco, mewah formal |

Data untuk template (lihat `services/invitation/renderer/view.go`): `.C` (konten), `.Couple` (urut sesuai `couple_order`), `.Events` (sudah diformat), `.FirstEvent`, `.Guest`, `.GuestName`, `.InvitationID`, `.Preview`, `.Theme`, `.Shared`, `.URL`.

---

## 8. Akses tamu — link `/nama-tamu` tervalidasi

- Setiap tamu punya `slug` dari nama (`Bapak Joko & Keluarga` → `bapak-joko-keluarga`), unik per undangan
  (bentrok → `-2`, `-3`). Slug **tidak berubah** walau nama diedit (link yang sudah disebar tetap hidup).
- Link personal: `https://<sub>.<base>/<slug>`. Kode 6 karakter (`/<KODE>`) tetap diterima (dipakai QR & passcode).
- Path yang tidak cocok dengan tamu mana pun → halaman **"Nama tidak terdaftar"** (HTTP 404), bukan undangan.
- `invitations.access_mode`:
  | Mode | `/` (tanpa nama) | `?to=Nama` | `/<slug>` | RSVP / ucapan / bukti hadiah |
  |---|---|---|---|---|
  | `public` (default) | undangan umum | nama ditampilkan (tidak divalidasi) | tervalidasi | siapa saja |
  | `guest_only` | **halaman gerbang** (input kode undangan) | diabaikan | tervalidasi | wajib `guest_code` valid |
- Keterbatasan: slug nama bisa ditebak. Untuk acara yang benar-benar tertutup, gunakan `guest_only` + check-in (QR/passcode di venue).

## 9. Check-in venue (fitur tambahan, opsional)

- Tersedia bila paket punya `allow_checkin = true` (admin boleh mengaktifkan tanpa syarat paket).
- Pengaturan undangan: `checkin_enabled`, `checkin_pin` (6–8 digit, disimpan bcrypt).
- Bila aktif, halaman tamu personal menampilkan **tiket**: QR (isi: `<link-tamu>?c=<KODE>`) + passcode `<KODE>` + jumlah orang.
- **Stasiun penerima tamu** (tanpa akun): `https://app.<base>/checkin/<invitation_id>` → masukkan PIN → token stasiun
  (JWT terpisah, issuer `undangan-checkin`, 12 jam, batal otomatis bila PIN diganti). Pemilik/admin yang login tidak perlu PIN.
- Scan QR (kamera) atau ketik passcode / cari nama → hasil:
  - `checked_in` ✅ nama, grup, jumlah orang diundang (penerima bisa isi jumlah yang datang, maks = pax undangan)
  - `already` ⚠️ sudah check-in pada jam X (tidak dihitung dua kali)
  - `not_found` ⛔ tidak terdaftar → tidak boleh masuk
  - `disabled` fitur check-in nonaktif
- Setiap percobaan dicatat di `checkin_logs` (termasuk yang ditolak). Pemilik bisa membatalkan check-in.
- Satu check-in per tamu per undangan (belum per sesi acara).

**Laporan kehadiran** (pemilik & admin): ringkasan (tamu/pax diundang, RSVP hadir, check-in tamu/pax, belum datang, persentase),
daftar tamu + status RSVP + waktu check-in, filter & export Excel.

## 10. Konfirmasi hadiah (bukti transfer / kado)

- Aktif bila `content.gift.enabled && content.gift.confirmation_enabled`.
- Tamu mengirim form di halaman undangan: nama, jenis (`transfer` | `kado`), tujuan (label rekening/e-wallet, opsional),
  nominal (opsional, transfer), pesan, **foto bukti** (jpg/png/webp ≤ 5 MB, wajib), `guest_code` (wajib di `guest_only`).
- File disimpan **privat** di object storage (`private/gifts/<invitation_id>/…`); hanya pemilik & admin yang bisa melihat.
- Pemilik: daftar + ringkasan (jumlah konfirmasi, total nominal, yang sudah diverifikasi), lihat bukti, tandai
  "sudah diterima", hapus, export Excel.
- Anti-spam: rate limit 5 kiriman / 10 menit per IP per undangan.

## 11. Object storage

`STORAGE_DRIVER=local|s3`. Driver `s3` kompatibel S3 (Cloudflare R2, AWS S3, MinIO, IDCloudHost, Biznet NEO, dsb.):
`S3_ENDPOINT, S3_REGION, S3_BUCKET, S3_ACCESS_KEY, S3_SECRET_KEY, S3_USE_SSL, S3_PUBLIC_BASE_URL`.
Kunci objek: `public/<yyyy>/<mm>/<acak>.<ext>` dan `private/<folder>/<acak>.<ext>`. File publik disajikan lewat
`S3_PUBLIC_BASE_URL` (CDN / bucket publik) bila diisi, jika tidak diproksikan lewat `/uploads/…`. File privat selalu
lewat API yang memeriksa hak akses.

## 12. API tambahan

| Method | Path | Akses | Ket |
|---|---|---|---|
| PUT | `/invitations/{id}/settings` | pemilik/admin | `{access_mode, checkin_enabled, checkin_pin?}` → Invitation |
| GET | `/invitations/{id}/attendance?q=&status=all\|checked_in\|not_checked_in&page=` | pemilik/admin | → `AttendanceReport` |
| GET | `/invitations/{id}/attendance/export.xlsx` | pemilik/admin | |
| POST | `/invitations/{id}/guests/{gid}/checkin` `{pax?}` / DELETE (batal) | pemilik/admin | → Guest |
| POST | `/checkin/{invitation_id}/session` `{pin, station_name?}` | publik, rate limit | → `{token, expires_at, invitation:{id,title,event_date}}` |
| GET | `/checkin/{invitation_id}/summary` | token stasiun / pemilik | → `CheckinSummary` |
| GET | `/checkin/{invitation_id}/guests?q=` | token stasiun / pemilik | → `{items: CheckinGuest[]}` (maks 20) |
| POST | `/checkin/{invitation_id}/scan` `{code, pax?}` | token stasiun / pemilik | → `CheckinResult` (`code` boleh berisi URL QR) |
| GET | `/checkin/{invitation_id}/recent` | token stasiun / pemilik | → `{items: CheckinLog[]}` (20 terakhir) |
| POST | `/public/invitations/{id}/gifts` | publik, multipart | → `{id, created_at}` |
| GET | `/invitations/{id}/gifts?page=` | pemilik/admin | → `GiftList` |
| GET | `/invitations/{id}/gifts/{gid}/proof` | pemilik/admin | gambar |
| PATCH / DELETE | `/invitations/{id}/gifts/{gid}` `{is_verified}` | pemilik/admin | → GiftConfirmation |
| GET | `/invitations/{id}/gifts/export.xlsx` | pemilik/admin | |

Token stasiun dikirim sebagai `Authorization: Bearer <token>`.

## 13. Domain sendiri (semua paket dengan `allow_custom_domain`, bawaan: Basic & Premium)

- Customer membeli domain sendiri di registrar, lalu di portal (tab Bagikan) memasukkan hostname.
- Backend membuat token dan menampilkan record DNS:
  - `TXT _undangan-verify.<hostname>` = token (bukti kepemilikan)
  - subdomain (mis. `www.rakadannadia.com`) → `CNAME` ke `CUSTOM_DOMAIN_CNAME`
  - domain utama (apex) → `A` ke `CUSTOM_DOMAIN_IPS` (bila diisi) atau CNAME flattening/ALIAS
- "Cek Verifikasi" memeriksa TXT + arah domain (CNAME cocok, atau IP domain = IP server/target). Terverifikasi →
  Caddy on-demand TLS menerbitkan SSL otomatis saat domain pertama kali dibuka; link tamu memakai domain sendiri.
- Subdomain gratis tetap aktif.

| Method | Path | Ket |
|---|---|---|
| PUT | `/invitations/{id}/custom-domain` `{hostname}` | → Invitation (409 `domain_taken`, 403 `feature_unavailable`) |
| POST | `/invitations/{id}/custom-domain/verify` | → Invitation (422 `dns_not_ready` + pesan yang kurang) |
| DELETE | `/invitations/{id}/custom-domain` | → Invitation |

`Invitation.custom_domain = {hostname, verified, verified_at, dns: [{type, name, value, purpose}]} | null`,
`Invitation.settings.custom_domain_available`.

## 14. Landing page & katalog tema

- Host platform (`<BASE_DOMAIN>`, dev `http://localhost:8080`) = landing page marketing (Go template, SEO): fitur, katalog tema
  (kategori + thumbnail + demo), harga dari tabel `plans`, kontak WhatsApp admin (dari pengaturan pembayaran), FAQ.
- `/tema` = katalog lengkap. CTA → `PORTAL_URL/daftar?tema=<slug>`.
- Tema: `theme.json.category` ∈ adat, islami, floral, modern, elegan, rustic, pastel, retro; thumbnail `themes/<slug>/assets/thumb.webp`
  (dibuat dengan `node tools/themeshot.mjs thumb <slug>`).

## 15. Notifikasi Telegram (opsional)

Aktif bila `TELEGRAM_BOT_TOKEN` & `TELEGRAM_CHAT_IDS` diisi. Port `kernel/notify.Notifier` dipakai service lain,
sehingga saluran lain (email/WA) tinggal menambah implementasi.

| Peristiwa | Isi pesan | Tombol |
|---|---|---|
| Customer baru mendaftar | nama, email, WhatsApp | – |
| Order dibuat | kode order, customer, paket, nominal + kode unik | – |
| Bukti transfer diunggah | idem + ajakan cek mutasi | ✅ Setujui · ⛔ Tolak |
| Order disetujui | pelaku, masa aktif langganan | – |
| Order ditolak | pelaku, alasan | – |

Perintah bot: `/start` atau `/id` (menampilkan chat id, boleh dari siapa saja), `/pending` (daftar order menunggu konfirmasi, khusus chat terdaftar).

Keamanan:
- Hanya chat di `TELEGRAM_CHAT_IDS` yang boleh menekan tombol; chat lain diabaikan tanpa balasan.
- Webhook `POST /api/v1/telegram/webhook` diverifikasi header `X-Telegram-Bot-Api-Secret-Token` (dikecualikan dari CSRF guard karena tidak memakai cookie sesi).
- Aksi bot dicatat di audit log sebagai super admin dengan nama pelaku Telegram.
- Notifikasi dikirim asinkron; kegagalan Telegram tidak pernah menggagalkan order/pembayaran.
