package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string // listener publik (di belakang Caddy)
	InternalAddr string // listener internal, loopback only
	BaseDomain   string // mis. "undangin.id"; dev: "localhost" → raka-nadia.localhost:8080
	// PublicURLPattern membentuk URL undangan; {sub} diganti subdomain.
	// dev: http://{sub}.localhost:8080 · prod: https://{sub}.undangin.id
	PublicURLPattern string
	DatabaseURL      string
	ThemesDir        string
	UploadDir        string // file publik (/uploads/...)
	PrivateDir       string // bukti bayar, tidak di-serve langsung

	// Object storage (docs/SPEC.md §11): local | s3
	StorageDriver   string
	S3Endpoint      string
	S3Region        string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3UseSSL        bool
	S3PathStyle     bool
	S3PublicBaseURL string
	Dev             bool

	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration

	// Hasil build SPA yang dilayani Go di admin.<base> dan app.<base>
	AdminDist  string
	LandingDir string // template landing page (web/landing)
	PortalDist string

	// Branding & URL (landing page, email, WA)
	AppName   string // nama brand, mis. "Undangin"
	PortalURL string // portal customer, mis. https://app.undangin.id
	AdminURL  string // portal admin, mis. https://admin.undangin.id (tautan di notifikasi)
	// Domain sendiri: customer mengarahkan CNAME ke target ini atau A record ke IP server.
	CustomDomainCNAME string
	CustomDomainIPs   []string

	// Notifikasi Telegram (opsional)
	TelegramBotToken      string
	TelegramChatIDs       []int64
	TelegramWebhookSecret string
	TelegramWebhookURL    string // kosong = polling (dev)
	TelegramAPIBase       string // kosong = api.telegram.org (diisi saat uji lokal)

	SeedAdminEmail    string
	SeedAdminPassword string
	SeedAdminName     string
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		Addr:                  env("API_ADDR", ":8080"),
		InternalAddr:          env("API_INTERNAL_ADDR", "127.0.0.1:8081"),
		BaseDomain:            env("BASE_DOMAIN", "localhost"),
		PublicURLPattern:      env("PUBLIC_URL_PATTERN", "http://{sub}.localhost:8080"),
		DatabaseURL:           env("DATABASE_URL", ""),
		ThemesDir:             env("THEMES_DIR", "themes"),
		UploadDir:             env("UPLOAD_DIR", "storage/uploads"),
		PrivateDir:            env("PRIVATE_DIR", "storage/private"),
		Dev:                   env("APP_ENV", "development") == "development",
		StorageDriver:         env("STORAGE_DRIVER", "local"),
		S3Endpoint:            env("S3_ENDPOINT", ""),
		S3Region:              env("S3_REGION", ""),
		S3Bucket:              env("S3_BUCKET", ""),
		S3AccessKey:           env("S3_ACCESS_KEY", ""),
		S3SecretKey:           env("S3_SECRET_KEY", ""),
		S3UseSSL:              env("S3_USE_SSL", "true") == "true",
		S3PathStyle:           env("S3_PATH_STYLE", "false") == "true",
		S3PublicBaseURL:       env("S3_PUBLIC_BASE_URL", ""),
		JWTSecret:             env("JWT_SECRET", ""),
		AccessTTL:             duration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:            duration("JWT_REFRESH_TTL", 30*24*time.Hour),
		AdminDist:             env("ADMIN_DIST", "../frontend/apps/admin/dist"),
		LandingDir:            env("LANDING_DIR", "web/landing"),
		PortalDist:            env("PORTAL_DIST", "../frontend/apps/portal/dist"),
		AppName:               env("APP_NAME", "Undangin"),
		PortalURL:             strings.TrimRight(env("PORTAL_URL", "http://localhost:5173"), "/"),
		CustomDomainCNAME:     env("CUSTOM_DOMAIN_CNAME", "custom."+env("BASE_DOMAIN", "localhost")),
		CustomDomainIPs:       splitList(env("CUSTOM_DOMAIN_IPS", "")),
		TelegramBotToken:      env("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatIDs:       splitInt64(env("TELEGRAM_CHAT_IDS", "")),
		TelegramWebhookSecret: env("TELEGRAM_WEBHOOK_SECRET", ""),
		TelegramWebhookURL:    env("TELEGRAM_WEBHOOK_URL", ""),
		TelegramAPIBase:       env("TELEGRAM_API_BASE", ""),
		SeedAdminEmail:        env("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword:     env("SEED_ADMIN_PASSWORD", ""),
		SeedAdminName:         env("SEED_ADMIN_NAME", "Super Admin"),
	}
}

func (c Config) InvitationURL(sub string) string {
	return strings.ReplaceAll(c.PublicURLPattern, "{sub}", sub)
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitInt64(s string) []int64 {
	var out []int64
	for _, p := range splitList(s) {
		if n, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func duration(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(env(key, "")); err == nil && d > 0 {
		return d
	}
	return fallback
}

// loadDotEnv mengisi env dari file KEY=VALUE tanpa menimpa env yang sudah ada.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`)
		if _, exists := os.LookupEnv(k); !exists {
			_ = os.Setenv(k, v)
		}
	}
}
