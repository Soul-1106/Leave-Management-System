package middleware

import (
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type jwtClaims struct {
	Subject   string          `json:"sub"`
	Email     string          `json:"email"`
	Issuer    string          `json:"iss"`
	Audience  json.RawMessage `json:"aud"`
	Expires   int64           `json:"exp"`
	NotBefore int64           `json:"nbf"`
}

type jwk struct {
	KeyID string `json:"kid"`
	Type  string `json:"kty"`
	Use   string `json:"use"`
	N     string `json:"n"`
	E     string `json:"e"`
}

var signingKeys = struct {
	sync.Mutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}{}

func verifySupabaseToken(ctx context.Context, token string) (string, string, error) {
	claims, err := verifyJWTLocally(ctx, token)
	if err == nil {
		return claims.Subject, claims.Email, nil
	}
	// Legacy projects may use an inaccessible symmetric signing secret. The
	// authenticated Supabase endpoint remains a secure compatibility fallback.
	return verifySupabaseTokenRemote(ctx, token)
}

func verifyJWTLocally(ctx context.Context, token string) (jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return jwtClaims{}, errors.New("malformed JWT")
	}
	var header jwtHeader
	var claims jwtClaims
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return claims, err
	}
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return claims, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	switch header.Algorithm {
	case "HS256":
		secret := os.Getenv("SUPABASE_JWT_SECRET")
		if secret == "" {
			return claims, errors.New("legacy JWT secret is unavailable")
		}
		expected := hmac.New(sha256.New, []byte(secret))
		_, _ = expected.Write([]byte(parts[0] + "." + parts[1]))
		if !hmac.Equal(signature, expected.Sum(nil)) {
			return claims, errors.New("invalid JWT signature")
		}
	case "RS256":
		key, err := supabaseSigningKey(ctx, header.KeyID)
		if err != nil {
			return claims, err
		}
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature); err != nil {
			return claims, errors.New("invalid JWT signature")
		}
	default:
		return claims, errors.New("unsupported JWT algorithm")
	}
	baseURL := strings.TrimRight(firstEnv("SUPABASE_URL", "VITE_SUPABASE_URL"), "/")
	now := time.Now().Unix()
	if claims.Subject == "" || claims.Expires <= now || claims.NotBefore > now+30 ||
		claims.Issuer != baseURL+"/auth/v1" || !hasAudience(claims.Audience, "authenticated") {
		return claims, errors.New("invalid JWT claims")
	}
	return claims, nil
}

func decodeJWTPart(value string, destination any) error {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, destination)
}

func hasAudience(raw json.RawMessage, expected string) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return single == expected
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) == nil {
		for _, audience := range multiple {
			if audience == expected {
				return true
			}
		}
	}
	return false
}

func supabaseSigningKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	signingKeys.Lock()
	defer signingKeys.Unlock()
	if time.Now().Before(signingKeys.expiresAt) && signingKeys.keys[keyID] != nil {
		return signingKeys.keys[keyID], nil
	}
	baseURL := strings.TrimRight(firstEnv("SUPABASE_URL", "VITE_SUPABASE_URL"), "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/auth/v1/.well-known/jwks.json", nil)
	if err != nil {
		return nil, err
	}
	response, err := authHTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("unable to load JWT signing keys")
	}
	var document struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return nil, err
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, item := range document.Keys {
		if item.Type != "RSA" || item.N == "" || item.E == "" {
			continue
		}
		modulus, nErr := base64.RawURLEncoding.DecodeString(item.N)
		exponent, eErr := base64.RawURLEncoding.DecodeString(item.E)
		if nErr != nil || eErr != nil {
			continue
		}
		e := 0
		for _, value := range exponent {
			e = e<<8 + int(value)
		}
		keys[item.KeyID] = &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: e}
	}
	signingKeys.keys = keys
	signingKeys.expiresAt = time.Now().Add(time.Hour)
	if keys[keyID] == nil {
		return nil, errors.New("JWT signing key not found")
	}
	return keys[keyID], nil
}
