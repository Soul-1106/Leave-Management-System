package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"leave-management-backend/internal/database"
)

const (
	SessionCookieName = "lms_session"
	CSRFCookieName    = "lms_csrf"
)

type Identity struct {
	UserID      string `json:"userId"`
	EmployeeID  string `json:"employeeId,omitempty"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Department  string `json:"department,omitempty"`
	Designation string `json:"designation,omitempty"`
}

type identityKey struct{}

var authHTTPClient = &http.Client{Timeout: 10 * time.Second}

func IdentityFrom(r *http.Request) (Identity, bool) {
	value, ok := r.Context().Value(identityKey{}).(Identity)
	return value, ok
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		token, err := requestToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		userID, email, err := verifySupabaseToken(r.Context(), token)
		if err != nil {
			http.Error(w, "invalid or expired access token", http.StatusUnauthorized)
			return
		}
		if database.DB == nil {
			http.Error(w, "database is not configured", http.StatusServiceUnavailable)
			return
		}

		var identity Identity
		identity.UserID, identity.Email = userID, email
		err = database.DB.QueryRowContext(r.Context(), `
			SELECT u.full_name, u.role::text, COALESCE(e.employee_id, ''),
			       COALESCE(d.name, ''), COALESCE(e.designation, '')
			FROM users u LEFT JOIN employees e ON e.user_id=u.id
			LEFT JOIN departments d ON d.id=e.department_id
			WHERE u.id=$1`, userID).Scan(&identity.Name, &identity.Role, &identity.EmployeeID, &identity.Department, &identity.Designation)
		if errors.Is(err, sql.ErrNoRows) {
			err = provisionEmployeeProfile(r.Context(), identity)
			if err == nil {
				err = database.DB.QueryRowContext(r.Context(), `
					SELECT u.full_name, u.role::text, COALESCE(e.employee_id, ''),
					       COALESCE(d.name, ''), COALESCE(e.designation, '')
					FROM users u LEFT JOIN employees e ON e.user_id=u.id
					LEFT JOIN departments d ON d.id=e.department_id
					WHERE u.id=$1`, userID).Scan(&identity.Name, &identity.Role, &identity.EmployeeID, &identity.Department, &identity.Designation)
			}
		}
		if err != nil {
			http.Error(w, "unable to load application profile", http.StatusServiceUnavailable)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, identity)))
	})
}

func requestToken(r *http.Request) (string, error) {
	if header := r.Header.Get("Authorization"); header != "" {
		return bearerToken(header)
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", errors.New("missing authentication session")
	}
	return cookie.Value, nil
}

// SessionHandler exchanges a verified Supabase access token for an HttpOnly,
// same-site application cookie. The token remains authoritative and is
// revalidated by AuthMiddleware on protected requests.
func SessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if _, _, err = verifySupabaseToken(r.Context(), token); err != nil {
		http.Error(w, "invalid or expired access token", http.StatusUnauthorized)
		return
	}
	csrf, err := randomToken(32)
	if err != nil {
		http.Error(w, "unable to create session", http.StatusInternalServerError)
		return
	}
	secure := cookieSecure(r)
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookieName, Value: token, Path: "/api",
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
		MaxAge: 3600,
	})
	http.SetCookie(w, &http.Cookie{
		Name: CSRFCookieName, Value: csrf, Path: "/",
		HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode,
		MaxAge: 3600,
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"csrfToken": csrf})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	secure := cookieSecure(r)
	for _, cookie := range []*http.Cookie{
		{Name: SessionCookieName, Path: "/api", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1},
		{Name: CSRFCookieName, Path: "/", HttpOnly: false, Secure: secure, SameSite: http.SameSiteLaxMode, MaxAge: -1},
	} {
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(http.StatusNoContent)
}

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// Bearer requests are not cookie-authenticated and are not vulnerable to
		// browser cross-site request forgery.
		if r.Header.Get("Authorization") != "" {
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" || r.Header.Get("X-CSRF-Token") != cookie.Value {
			http.Error(w, "invalid CSRF token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func cookieSecure(r *http.Request) bool {
	if value := strings.ToLower(os.Getenv("COOKIE_SECURE")); value != "" {
		return value != "false" && value != "0"
	}
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func provisionEmployeeProfile(ctx context.Context, identity Identity) error {
	tx, err := database.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	name := strings.Split(identity.Email, "@")[0]
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO users(id, email, full_name, role)
		VALUES($1, $2, $3, 'employee')
		ON CONFLICT(id) DO NOTHING`, identity.UserID, identity.Email, name); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO employees(user_id, employee_id, designation, joining_date)
		VALUES($1, 'EMP-' || upper(substr(replace($1::text, '-', ''), 1, 8)), 'Employee', CURRENT_DATE)
		ON CONFLICT(user_id) DO NOTHING`, identity.UserID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func RequireRoles(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := IdentityFrom(r)
			if !ok {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if !allowed[identity.Role] {
				http.Error(w, "insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", errors.New("missing bearer access token")
	}
	return parts[1], nil
}

func verifySupabaseTokenRemote(ctx context.Context, token string) (string, string, error) {
	baseURL := strings.TrimRight(firstEnv("SUPABASE_URL", "VITE_SUPABASE_URL"), "/")
	apiKey := firstEnv("SUPABASE_ANON_KEY", "VITE_SUPABASE_ANON_KEY")
	if baseURL == "" || apiKey == "" {
		return "", "", errors.New("Supabase Auth is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/auth/v1/user", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", apiKey)
	res, err := authHTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", "", errors.New("token rejected")
	}
	var user struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil || user.ID == "" {
		return "", "", errors.New("invalid user response")
	}
	return user.ID, user.Email, nil
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
