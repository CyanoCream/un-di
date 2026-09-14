# Undangan Digital

Folder kerja berisi **2 repo git terpisah**:

| Repo | Isi |
|---|---|
| [`backend/`](backend/README.md) | Go workspace: API, landing page, tema undangan, migrasi & script server |
| [`frontend/`](frontend/README.md) | Vue 3: portal super admin, portal customer, package shared |

Saat development jalankan keduanya berdampingan (backend :8080, portal :5173, admin :5174).
Saat deploy, backend melayani hasil build frontend berdasarkan subdomain — satu server.
