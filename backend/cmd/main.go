package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"leave-management-backend/internal/database"
	"leave-management-backend/internal/handlers"
	"leave-management-backend/internal/middleware"
)

//go:embed web
var frontendFiles embed.FS

func main() {
	loadEnvFile()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer startupCancel()
	if err := database.InitDB(startupCtx); err != nil {
		log.Fatalf("database_startup_failed error=%q", err)
	}

	// API Routes (These require authentication)
	apiMux := http.NewServeMux()
	responseCache := middleware.NewResponseCache(durationEnv("CACHE_TTL_SECONDS", 15) * time.Second)
	apiLimiter := middleware.RateLimitFromEnv("RATE_LIMIT_PER_MINUTE", 120, time.Minute)
	authLimiter := middleware.RateLimitFromEnv("AUTH_RATE_LIMIT_PER_MINUTE", 10, time.Minute)
	authenticated := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(handler)
	}
	authenticatedCached := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(responseCache.Middleware(handler))
	}
	managerCached := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("manager", "admin")(responseCache.Middleware(handler)))
	}
	managerOnly := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("manager", "admin")(handler))
	}
	employeeCached := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("employee")(responseCache.Middleware(handler)))
	}
	employeeWrite := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("employee")(
			middleware.CSRFMiddleware(responseCache.InvalidateOnSuccess(handler)),
		))
	}
	managerWrite := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("manager", "admin")(
			middleware.CSRFMiddleware(responseCache.InvalidateOnSuccess(handler)),
		))
	}
	adminCached := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("admin")(responseCache.Middleware(handler)))
	}
	adminWrite := func(handler http.HandlerFunc) http.Handler {
		return middleware.AuthMiddleware(middleware.RequireRoles("admin")(
			middleware.CSRFMiddleware(responseCache.InvalidateOnSuccess(handler)),
		))
	}

	apiMux.Handle("/api/me", authenticated(http.HandlerFunc(handlers.GetMe)))
	apiMux.Handle("/api/session", authLimiter.Middleware(http.HandlerFunc(middleware.SessionHandler)))
	apiMux.Handle("/api/session/logout", middleware.CSRFMiddleware(http.HandlerFunc(middleware.LogoutHandler)))
	apiMux.Handle("/api/dashboard/stats", authenticatedCached(http.HandlerFunc(handlers.GetDashboardStats)))
	apiMux.Handle("/api/leaves/my", methodSplit(
		employeeCached(http.HandlerFunc(handlers.GetMyLeaves)),
		employeeWrite(http.HandlerFunc(handlers.GetMyLeaves)),
	))
	apiMux.Handle("/api/leaves/my/", employeeWrite(http.HandlerFunc(handlers.DeleteMyLeave)))
	apiMux.Handle("/api/leaves/approvals", managerCached(http.HandlerFunc(handlers.GetApprovals)))
	apiMux.Handle("/api/leaves/history", managerCached(http.HandlerFunc(handlers.GetApprovalHistory)))
	apiMux.Handle("/api/employees", managerCached(http.HandlerFunc(handlers.GetEmployees)))
	apiMux.Handle("/api/leaves/balances", authenticatedCached(http.HandlerFunc(handlers.GetLeaveBalances)))
	apiMux.Handle("/api/leaves/", managerWrite(http.HandlerFunc(handlers.DecideLeave)))
	apiMux.Handle("/api/attachments/", methodSplit(
		managerOnly(http.HandlerFunc(handlers.GetAttachment)),
		employeeWrite(http.HandlerFunc(handlers.UploadAttachment)),
	))
	apiMux.Handle("/api/admin/departments", adminCached(http.HandlerFunc(handlers.AdminDepartments)))
	apiMux.Handle("/api/admin/balances", methodSplit(
		adminCached(http.HandlerFunc(handlers.AdminBalances)),
		adminWrite(http.HandlerFunc(handlers.AdminBalances)),
	))
	apiMux.Handle("/api/admin/users", methodSplit(
		adminCached(http.HandlerFunc(handlers.AdminUsers)),
		adminWrite(http.HandlerFunc(handlers.AdminUsers)),
	))
	apiMux.Handle("/api/admin/users/", adminWrite(http.HandlerFunc(handlers.AdminUser)))

	webFiles, err := fs.Sub(frontendFiles, "web")
	if err != nil {
		log.Fatalf("Failed to load embedded frontend: %v", err)
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/api/", apiLimiter.Middleware(apiMux))
	rootMux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	rootMux.HandleFunc("/health/ready", handlers.Readiness)
	rootMux.Handle("/", spaHandler(http.FS(webFiles)))

	// Apply CORS Middleware to the entire server
	handler := middleware.RequestLogger(middleware.CORSMiddleware(rootMux))

	log.Printf("Server starting on port %s...\n", port)
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Shutdown signal received; draining connections...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("server_shutdown_failed error=%q", err)
	}
	if database.DB != nil {
		_ = database.DB.Close()
	}
}

func loadEnvFile() {
	candidates := []string{".env", filepath.Join("..", ".env")}
	for _, name := range candidates {
		content, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), `"'`)
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
		return
	}
}

func durationEnv(name string, fallback int) time.Duration {
	if value := os.Getenv(name); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			return time.Duration(parsed)
		}
	}
	return time.Duration(fallback)
}

func methodSplit(read, write http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			read.ServeHTTP(w, r)
			return
		}
		write.ServeHTTP(w, r)
	})
}

func spaHandler(files http.FileSystem) http.Handler {
	fileServer := http.FileServer(files)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if file, err := files.Open(r.URL.Path); err == nil {
				_ = file.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
