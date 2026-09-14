# Undangan Digital (un-di)

Platform undangan pernikahan digital. Satu repo, dua bagian yang berdiri sendiri:

| Folder | Isi |
|---|---|
| [`backend/`](backend/README.md) | Go workspace (DDD, siap microservice): API, landing page, 20 tema undangan, migrasi & script server |
| [`frontend/`](frontend/README.md) | Vue 3: portal super admin, portal customer (+ stasiun check-in), package shared |

- Development: backend `:8080`, portal `:5173`, admin `:5174` (lihat README masing-masing).
- Deploy VPS: [`backend/docs/DEPLOY.md`](backend/docs/DEPLOY.md) — backend melayani hasil build frontend per subdomain, satu server.
