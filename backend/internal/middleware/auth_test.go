package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerToken(t *testing.T) {
	token, err := bearerToken("Bearer signed-token")
	if err != nil || token != "signed-token" {
		t.Fatalf("got token %q and error %v", token, err)
	}
	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer one two"} {
		if _, err := bearerToken(header); err == nil {
			t.Fatalf("expected header %q to be rejected", header)
		}
	}
}

func TestCSRFMiddlewareAcceptsMatchingCookieAndHeader(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPatch, "/api/leaves/id/decision", nil)
	request.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "matching-token"})
	request.Header.Set("X-CSRF-Token", "matching-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("got status %d, called=%v", response.Code, called)
	}
}

func TestCSRFMiddlewareAllowsBearerRequest(t *testing.T) {
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodPost, "/api/leaves/my", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("got %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequireRoles(t *testing.T) {
	handler := RequireRoles("admin")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	tests := []struct {
		name     string
		identity *Identity
		status   int
	}{
		{name: "missing identity", status: http.StatusUnauthorized},
		{name: "wrong role", identity: &Identity{UserID: "employee", Role: "employee"}, status: http.StatusForbidden},
		{name: "allowed role", identity: &Identity{UserID: "admin", Role: "admin"}, status: http.StatusNoContent},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
			if test.identity != nil {
				request = request.WithContext(context.WithValue(request.Context(), identityKey{}, *test.identity))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("got %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestCookieSecureConfiguration(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://example.test", nil)
	t.Setenv("COOKIE_SECURE", "true")
	if !cookieSecure(request) {
		t.Fatal("COOKIE_SECURE=true should force secure cookies")
	}
	t.Setenv("COOKIE_SECURE", "false")
	if cookieSecure(request) {
		t.Fatal("COOKIE_SECURE=false should allow local HTTP cookies")
	}
}
