package middleware

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type rateEntry struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]rateEntry
	limit   int
	window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{entries: make(map[string]rateEntry), limit: limit, window: window}
}

func (limiter *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		key := clientIP(r)
		limiter.mu.Lock()
		entry := limiter.entries[key]
		if entry.resetAt.IsZero() || now.After(entry.resetAt) {
			entry = rateEntry{resetAt: now.Add(limiter.window)}
		}
		entry.count++
		limiter.entries[key] = entry
		remaining := limiter.limit - entry.count
		if remaining < 0 {
			remaining = 0
		}
		if len(limiter.entries) > 10000 {
			for candidate, value := range limiter.entries {
				if now.After(value.resetAt) {
					delete(limiter.entries, candidate)
				}
			}
		}
		limiter.mu.Unlock()

		w.Header().Set("RateLimit-Limit", strconv.Itoa(limiter.limit))
		w.Header().Set("RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(entry.resetAt.Unix(), 10))
		if entry.count > limiter.limit {
			retry := int(time.Until(entry.resetAt).Seconds()) + 1
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if trustedProxy() {
		if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
			return forwarded
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func trustedProxy() bool {
	value, err := strconv.ParseBool(os.Getenv("TRUST_PROXY"))
	return err == nil && value
}

func RateLimitFromEnv(name string, defaultLimit int, window time.Duration) *RateLimiter {
	limit := defaultLimit
	if value := os.Getenv(name); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	return NewRateLimiter(limit, window)
}
