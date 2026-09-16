package httpx

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CSRFGuard: request non-GET wajib header X-Requested-With (form lintas situs tidak bisa menambahkannya).
// Dipasang di /api karena token disimpan di cookie HttpOnly.
//
// exempt = path yang dikecualikan (mis. webhook pihak ketiga yang punya pengaman sendiri dan
// tidak memakai cookie sesi, sehingga tidak bisa jadi sasaran CSRF).
func CSRFGuard(next http.Handler, exempt ...string) http.Handler {
	skip := make(map[string]bool, len(exempt))
	for _, p := range exempt {
		skip[p] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions:
		case skip[r.URL.Path]:
		default:
			if r.Header.Get("X-Requested-With") == "" && !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				JSON(w, http.StatusForbidden, map[string]any{"error": map[string]string{"code": "csrf", "message": "Header X-Requested-With wajib"}})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// AccessLog + recover panic.
func AccessLog(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "path", r.URL.Path, "panic", p)
				http.Error(rec, "internal error", http.StatusInternalServerError)
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rec.status, "ms", time.Since(start).Milliseconds())
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// RateLimiter: fixed window di memori (cukup untuk satu instance).
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*bucket
}

type bucket struct {
	start time.Time
	count int
}

func NewRateLimiter(limit int, per time.Duration) *RateLimiter {
	rl := &RateLimiter{limit: limit, window: per, hits: map[string]*bucket{}}
	go func() {
		for range time.Tick(per) {
			rl.mu.Lock()
			for k, b := range rl.hits {
				if time.Since(b.start) > rl.window {
					delete(rl.hits, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.hits[key]
	if !ok || time.Since(b.start) > rl.window {
		rl.hits[key] = &bucket{start: time.Now(), count: 1}
		return true
	}
	b.count++
	return b.count <= rl.limit
}
