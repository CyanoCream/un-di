#!/usr/bin/env bash
# Update aplikasi di VPS: tarik kode terbaru, build frontend, backup database, build & restart backend.
#
#   ./deploy/update.sh
set -euo pipefail

BACKEND="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND="$BACKEND/../frontend"
cd "$BACKEND/deploy"

echo "→ git pull"
git -C "$BACKEND" pull --ff-only
# frontend repo terpisah? tarik juga (bila satu repo, perintah ini tidak mengubah apa-apa)
if [ "$(git -C "$FRONTEND" rev-parse --show-toplevel)" != "$(git -C "$BACKEND" rev-parse --show-toplevel)" ]; then
  git -C "$FRONTEND" pull --ff-only
fi

echo "→ backup sebelum update"
docker compose exec -T app ./scripts/db-backup.sh

"$BACKEND/deploy/build-frontend.sh" "$FRONTEND"

echo "→ build & restart backend (migrasi database berjalan otomatis saat start)"
docker compose up -d --build app
docker compose ps
echo "Update selesai."
