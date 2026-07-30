package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestVerifyJWTLocallyWithLegacySecret(t *testing.T) {
	t.Setenv("SUPABASE_URL", "https://example.supabase.co")
	t.Setenv("SUPABASE_JWT_SECRET", "test-secret")
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(
		`{"sub":"user-1","email":"person@example.com","iss":"https://example.supabase.co/auth/v1","aud":"authenticated","exp":%d}`,
		time.Now().Add(time.Hour).Unix(),
	)))
	unsigned := header + "." + claims
	mac := hmac.New(sha256.New, []byte("test-secret"))
	_, _ = mac.Write([]byte(unsigned))
	token := unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	result, err := verifyJWTLocally(context.Background(), token)
	if err != nil {
		t.Fatalf("token was rejected: %v", err)
	}
	if result.Subject != "user-1" {
		t.Fatalf("got subject %q", result.Subject)
	}
}
