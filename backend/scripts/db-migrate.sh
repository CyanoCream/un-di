#!/usr/bin/env bash
# Jalankan migrasi SQL yang belum pernah dijalankan (pakai psql, tanpa Go).
# Aman diulang: versi yang sudah jalan dicatat di schema_migrations (sama dengan migrator bawaan server).
#
#   DATABASE_URL=postgres://user:pass@host:5432/undangan ./scripts/db-migrate.sh
#   ./scripts/db-migrate.sh --status
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -z "${DATABASE_URL:-}" && -f "$DIR/.env" ]]; then
  DATABASE_URL="$(grep -E '^DATABASE_URL=' "$DIR/.env" | head -1 | cut -d= -f2- | tr -d '"'"'")"
fi
: "${DATABASE_URL:?Isi DATABASE_URL (env atau file .env)}"
export PGOPTIONS="${PGOPTIONS:--c client_min_messages=warning}"
PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -X -q)

"${PSQL[@]}" -c "CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"

applied=0; pending=0
for file in "$DIR"/migrations/*.sql; do
  version="$(basename "$file" .sql)"
  done_already="$("${PSQL[@]}" -tAc "SELECT 1 FROM schema_migrations WHERE version = '$version'")"
  if [[ "$done_already" == "1" ]]; then
    [[ "${1:-}" == "--status" ]] && echo "[x] $version"
    applied=$((applied + 1)); continue
  fi
  if [[ "${1:-}" == "--status" ]]; then
    echo "[ ] $version"; pending=$((pending + 1)); continue
  fi
  echo "→ $version"
  # Satu file = satu transaksi; gagal → rollback, versi tidak dicatat.
  "${PSQL[@]}" --single-transaction \
    -f "$file" \
    -c "INSERT INTO schema_migrations (version) VALUES ('$version')"
  pending=$((pending + 1))
done

if [[ "${1:-}" == "--status" ]]; then
  echo "$applied sudah, $pending belum"
else
  echo "Selesai: $pending migrasi baru dijalankan ($applied sudah ada sebelumnya)."
fi
