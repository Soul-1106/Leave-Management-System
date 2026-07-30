package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestResponseCacheCachesPerUserAndInvalidates(t *testing.T) {
	cache := NewResponseCache(time.Minute)
	calls := 0
	handler := cache.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(strconv.Itoa(calls)))
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/value", nil)
	request = request.WithContext(context.WithValue(request.Context(), identityKey{}, Identity{UserID: "user-1"}))
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request)
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, request)

	if first.Body.String() != "1" || second.Body.String() != "1" || calls != 1 {
		t.Fatalf("cache miss/hit mismatch: first=%q second=%q calls=%d", first.Body.String(), second.Body.String(), calls)
	}
	if second.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("expected cache hit header, got %q", second.Header().Get("X-Cache"))
	}

	cache.Clear()
	third := httptest.NewRecorder()
	handler.ServeHTTP(third, request)
	if third.Body.String() != "2" || calls != 2 {
		t.Fatalf("cache was not invalidated: body=%q calls=%d", third.Body.String(), calls)
	}
}

func TestCSRFMiddlewareRejectsMismatchedToken(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/leaves/my", nil)
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "expected"})
	request.Header.Set("X-CSRF-Token", "wrong")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("got %d, want %d", response.Code, http.StatusForbidden)
	}
}
