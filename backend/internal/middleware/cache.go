package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"
)

type cachedResponse struct {
	body      []byte
	header    http.Header
	status    int
	expiresAt time.Time
	userID    string
	role      string
}

type ResponseCache struct {
	mu      sync.RWMutex
	entries map[string]cachedResponse
	ttl     time.Duration
}

func NewResponseCache(ttl time.Duration) *ResponseCache {
	return &ResponseCache{entries: make(map[string]cachedResponse), ttl: ttl}
}

func (cache *ResponseCache) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		identity, _ := IdentityFrom(r)
		key := identity.UserID + "|" + r.URL.RequestURI()
		now := time.Now()
		cache.mu.RLock()
		entry, found := cache.entries[key]
		cache.mu.RUnlock()
		if found && now.Before(entry.expiresAt) {
			copyHeaders(w.Header(), entry.header)
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(entry.status)
			_, _ = w.Write(entry.body)
			return
		}

		recorder := &cacheRecorder{ResponseWriter: w, status: http.StatusOK}
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Cache", "MISS")
		next.ServeHTTP(recorder, r)
		if recorder.status >= 200 && recorder.status < 300 {
			headers := recorder.Header().Clone()
			headers.Set("Cache-Control", "private, no-store")
			cache.mu.Lock()
			cache.entries[key] = cachedResponse{
				body: append([]byte(nil), recorder.body.Bytes()...), header: headers,
				status: recorder.status, expiresAt: now.Add(cache.ttl),
				userID: identity.UserID, role: identity.Role,
			}
			cache.mu.Unlock()
		}
	})
}

func (cache *ResponseCache) Clear() {
	cache.mu.Lock()
	cache.entries = make(map[string]cachedResponse)
	cache.mu.Unlock()
}

func (cache *ResponseCache) InvalidateOnSuccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if recorder.status >= 200 && recorder.status < 300 {
			identity, _ := IdentityFrom(r)
			cache.invalidateRelated(identity)
		}
	})
}

func (cache *ResponseCache) invalidateRelated(identity Identity) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	for key, entry := range cache.entries {
		// Always invalidate the caller. Employee changes can affect their manager
		// and admins; management changes can affect employee-facing data.
		if entry.userID == identity.UserID ||
			entry.role == "admin" ||
			(identity.Role == "employee" && entry.role == "manager") ||
			(identity.Role != "employee" && entry.role == "employee") {
			delete(cache.entries, key)
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

type cacheRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (recorder *cacheRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *cacheRecorder) Write(value []byte) (int, error) {
	_, _ = recorder.body.Write(value)
	return recorder.ResponseWriter.Write(value)
}

func copyHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}
