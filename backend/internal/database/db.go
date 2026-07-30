package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB(ctx context.Context) error {
	dbURL := os.Getenv("SUPABASE_DB_URL")
	if dbURL == "" {
		return fmt.Errorf("SUPABASE_DB_URL is not set")
	}

	var err error
	DB, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	DB.SetMaxOpenConns(intEnv("DB_MAX_OPEN_CONNS", 10))
	DB.SetMaxIdleConns(intEnv("DB_MAX_IDLE_CONNS", 5))
	DB.SetConnMaxLifetime(durationEnv("DB_CONN_MAX_LIFETIME_SECONDS", 1800))
	DB.SetConnMaxIdleTime(durationEnv("DB_CONN_MAX_IDLE_SECONDS", 300))

	delay := 500 * time.Millisecond
	for attempt := 1; attempt <= 5; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = DB.PingContext(pingCtx)
		cancel()
		if err == nil {
			log.Println("database_connected provider=supabase")
			return nil
		}
		if attempt < 5 {
			log.Printf("database_retry attempt=%d error=%q", attempt, err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay *= 2
			}
		}
	}
	_ = DB.Close()
	DB = nil
	return fmt.Errorf("ping database after retries: %w", err)
}

func intEnv(name string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(name)); err == nil && value > 0 {
		return value
	}
	return fallback
}

func durationEnv(name string, fallbackSeconds int) time.Duration {
	return time.Duration(intEnv(name, fallbackSeconds)) * time.Second
}
