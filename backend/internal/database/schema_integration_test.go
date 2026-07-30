package database

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestFreshSchemaObjectsExist(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_DB_INTEGRATION_TESTS=1 to test a configured Supabase database")
	}
	databaseURL := os.Getenv("SUPABASE_DB_URL")
	if databaseURL == "" {
		t.Fatal("SUPABASE_DB_URL is required for database integration tests")
	}
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tables int
	err = conn.QueryRowContext(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema='public'
		  AND table_name IN ('departments','users','employees','leave_types','leaves','leave_balances')
	`).Scan(&tables)
	if err != nil {
		t.Fatal(err)
	}
	if tables != 6 {
		t.Fatalf("found %d of 6 required application tables", tables)
	}

	var bucketExists bool
	if err = conn.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM storage.buckets WHERE id='leave-attachments')`,
	).Scan(&bucketExists); err != nil {
		t.Fatal(err)
	}
	if !bucketExists {
		t.Fatal("leave-attachments storage bucket is missing")
	}
}
