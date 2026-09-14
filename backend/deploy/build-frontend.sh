#!/usr/bin/env bash
# Build repo frontend memakai container Node (VPS tidak perlu install Node), lalu salin hasilnya ke deploy/dist.
#
#   ./deploy/build-frontend.sh [path-repo-frontend]      (default: ../frontend di samping repo backend)
#
# URL publik dibaca dari backend/.env: BASE_DOMAIN & APP_NAME.
set -euo pipefail

BACKEND="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND="$(cd "${1:-$BACKEND/../frontend}" && pwd)"
envval() { grep -E "^$1=" "$BACKEND/.env" 2>/dev/null | head -1 | cut -d= -f2- | tr -d '"'"'" || true; }
BASE_DOMAIN="$(envval BASE_DOMAIN)"; : "${BASE_DOMAIN:?Isi BASE_DOMAIN di backend/.env}"
APP_NAME="$(envval APP_NAME)"; APP_NAME="${APP_NAME:-Undangin}"

echo "→ build frontend ($FRONTEND) untuk https://$BASE_DOMAIN"
docker run --rm \
  -v "$FRONTEND":/w -w /w \
  -e VITE_LANDING_URL="https://$BASE_DOMAIN" \
  -e VITE_PORTAL_URL="https://app.$BASE_DOMAIN" \
  -e VITE_APP_NAME="$APP_NAME" \
  node:22-alpine sh -c "npm ci --no-audit --no-fund && npm run build"

echo "→ salin dist"
rm -rf "$BACKEND/deploy/dist"
mkdir -p "$BACKEND/deploy/dist"
cp -r "$FRONTEND/apps/admin/dist" "$BACKEND/deploy/dist/admin"
cp -r "$FRONTEND/apps/portal/dist" "$BACKEND/deploy/dist/portal"
echo "Selesai: deploy/dist/admin & deploy/dist/portal"
