# Image backend: API + landing page + halaman undangan (tema) dalam SATU binary.
# Build dari root repo backend:  docker build -f deploy/Dockerfile -t undangan-backend .
#
# Hasil build frontend (repo frontend: apps/admin/dist & apps/portal/dist) di-mount ke /app/dist/admin & /app/dist/portal
# (lihat deploy/docker-compose.yml), jadi update tampilan portal tidak perlu build ulang image ini.

FROM golang:1.26-alpine AS build
WORKDIR /src
# Salin go.mod/go.sum semua modul dulu supaya layer unduhan dependency ter-cache.
COPY go.work ./
COPY kernel/go.mod kernel/go.sum* kernel/
COPY migrations/go.mod migrations/
COPY app/go.mod app/go.sum* app/
COPY services/audit/go.mod services/audit/go.sum* services/audit/
COPY services/auth/go.mod services/auth/go.sum* services/auth/
COPY services/billing/go.mod services/billing/go.sum* services/billing/
COPY services/dashboard/go.mod services/dashboard/go.sum* services/dashboard/
COPY services/invitation/go.mod services/invitation/go.sum* services/invitation/
COPY services/landing/go.mod services/landing/go.sum* services/landing/
COPY services/media/go.mod services/media/go.sum* services/media/
COPY services/theme/go.mod services/theme/go.sum* services/theme/
COPY services/user/go.mod services/user/go.sum* services/user/
RUN cd app && go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./app/cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./app/cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata postgresql17-client bash && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /out/migrate ./
COPY themes ./themes
COPY web ./web
COPY migrations ./migrations
COPY scripts ./scripts
RUN mkdir -p storage backups dist/admin dist/portal && chown -R app /app/storage /app/backups
USER app
ENV APP_ENV=production \
    TZ=Asia/Jakarta \
    API_ADDR=:8080 \
    API_INTERNAL_ADDR=0.0.0.0:8081 \
    THEMES_DIR=themes \
    LANDING_DIR=web/landing \
    ADMIN_DIST=dist/admin \
    PORTAL_DIST=dist/portal \
    UPLOAD_DIR=storage/uploads \
    PRIVATE_DIR=storage/private
EXPOSE 8080
CMD ["./server"]
