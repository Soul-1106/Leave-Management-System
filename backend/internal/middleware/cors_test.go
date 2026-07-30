package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://leave.example.com, https://admin.example.com")
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://leave.example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Header().Get("Access-Control-Allow-Origin") != "https://leave.example.com" {
		t.Fatalf("configured origin was not allowed: %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentialed CORS header is missing")
	}
}

func TestCORSDoesNotReflectUnknownOrigin(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://leave.example.com")
	handler := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if origin := response.Header().Get("Access-Control-Allow-Origin"); origin != "" {
		t.Fatalf("unexpected reflected origin %q", origin)
	}
}
