#!/usr/bin/env bash
# Pulihkan backup di server baru.
#
#   DATABASE_URL=postgres://user:pass@host:5432/undangan ./scripts/db-restore.sh backups/undangan-20260913-221500
#
# PERINGATAN: objek database yang namanya sama akan DITIMPA (--clean). Pastikan DATABASE_URL menunjuk database yang benar.
set -euo pipefail

SRC="${1:?Pakai: db-restore.sh <folder-backup>}"
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
envval() { grep -E "^$1=" "$DIR/.env" 2>/dev/null | head -1 | cut -d= -f2- | tr -d '"'"'" || true; }
DATABASE_URL="${DATABASE_URL:-$(envval DATABASE_URL)}"
: "${DATABASE_URL:?Isi DATABASE_URL (env atau file .env)}"

[[ -f "$SRC/database.dump" ]] || { echo "database.dump tidak ditemukan di $SRC" >&2; exit 1; }
if [[ -f "$SRC/SHA256SUMS" ]]; then
  ( cd "$SRC" && sha256sum --check --quiet SHA256SUMS ) || { echo "Checksum backup tidak cocok!" >&2; exit 1; }
fi

host="$(printf '%s' "$DATABASE_URL" | sed -E 's#^[a-z]+://([^:@/]+(:[^@/]*)?@)?([^/:?]+).*#\3#')"
read -r -p "Restore ke database di host '$host'? Data lama dengan nama sama akan ditimpa. Ketik 'ya': " ok
[[ "$ok" == "ya" ]] || { echo "Dibatalkan."; exit 1; }

echo "→ restore database"
pg_restore --dbname "$DATABASE_URL" --clean --if-exists --no-owner --no-privileges --exit-on-error "$SRC/database.dump"

if [[ -f "$SRC/storage.tar.gz" ]]; then
  echo "→ ekstrak storage ke $DIR"
  tar -C "$DIR" -xzf "$SRC/storage.tar.gz"
fi

echo "→ jalankan migrasi terbaru (bila kode lebih baru dari backup)"
DATABASE_URL="$DATABASE_URL" "$DIR/scripts/db-migrate.sh"
echo "Restore selesai."
