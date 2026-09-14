#!/usr/bin/env bash
# Backup untuk pindah server: database (pg_dump format custom) + folder storage (foto, musik, bukti bayar/hadiah).
# Bila STORAGE_DRIVER=s3, file ada di object storage (tidak perlu dipindah) → hanya database yang di-backup.
#
#   ./scripts/db-backup.sh                → backups/undangan-YYYYmmdd-HHMMSS/
#   BACKUP_DIR=/mnt/backup ./scripts/db-backup.sh
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
envval() { grep -E "^$1=" "$DIR/.env" 2>/dev/null | head -1 | cut -d= -f2- | tr -d '"'"'" || true; }
DATABASE_URL="${DATABASE_URL:-$(envval DATABASE_URL)}"
: "${DATABASE_URL:?Isi DATABASE_URL (env atau file .env)}"
UPLOAD_DIR="${UPLOAD_DIR:-$(envval UPLOAD_DIR)}"; UPLOAD_DIR="${UPLOAD_DIR:-storage/uploads}"
PRIVATE_DIR="${PRIVATE_DIR:-$(envval PRIVATE_DIR)}"; PRIVATE_DIR="${PRIVATE_DIR:-storage/private}"
STORAGE_DRIVER="${STORAGE_DRIVER:-$(envval STORAGE_DRIVER)}"; STORAGE_DRIVER="${STORAGE_DRIVER:-local}"

OUT="${BACKUP_DIR:-$DIR/backups}/undangan-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUT"

echo "→ dump database"
pg_dump "$DATABASE_URL" --format=custom --no-owner --no-privileges --file "$OUT/database.dump"

if [[ "$STORAGE_DRIVER" == "local" ]]; then
  echo "→ arsip storage lokal"
  tar -C "$DIR" -czf "$OUT/storage.tar.gz" "$UPLOAD_DIR" "$PRIVATE_DIR" 2>/dev/null || \
    echo "  (folder storage belum ada, dilewati)"
fi

( cd "$OUT" && sha256sum ./* > SHA256SUMS )
echo "Backup selesai: $OUT"
ls -lh "$OUT"
