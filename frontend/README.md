# Undangan Digital — Frontend

Monorepo npm workspaces untuk dua SPA Vue 3 + Vite + Tailwind v4:

```
apps/
  admin/    Admin Console (super admin)      dev :5174   prod admin.<domain>
  portal/   Portal customer + stasiun check-in  dev :5173   prod app.<domain>
packages/
  shared/   types (kontrak API), API client (auto refresh JWT), komponen editor/tema/tamu/kehadiran/hadiah/domain
```

Kontrak API ada di repo backend: `docs/SPEC.md`. `packages/shared/src/types.ts` harus selalu sinkron dengannya.

## Development

Backend harus jalan di `http://localhost:8080` (repo backend). Vite mem-proxy `/api`, `/uploads`, `/_preview`, `/_theme`, `/_shared` ke sana.

```bash
npm install
npm run dev:portal   # http://localhost:5173
npm run dev:admin    # http://localhost:5174
npm run build        # type-check + build → apps/*/dist
```

Env (lihat `.env.example` tiap app): `VITE_LANDING_URL`, `VITE_APP_NAME` (portal), `VITE_PORTAL_URL` (admin).

## Deploy

Hasil `apps/admin/dist` dan `apps/portal/dist` disajikan backend Go berdasarkan subdomain
(`ADMIN_DIST`, `PORTAL_DIST`), jadi cukup salin kedua folder `dist` ke server backend.
Alternatif: host statis mana pun dengan fallback SPA ke `index.html`, selama `/api/*` diproksikan ke backend di origin yang sama
(cookie auth memakai SameSite + CSRF header).
