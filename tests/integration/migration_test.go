package integration

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestApplyMigrations loads SQL files from persistence/migrations and executes
// them against DATABASE_URL, then asserts that core tables exist.
func TestApplyMigrations(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping migration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	migDir := filepath.Join("persistence", "migrations")
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, n := range names {
		path := filepath.Join(migDir, n)
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", n, err)
		}
		sql := string(b)
		if sql == "" {
			continue
		}
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatalf("exec migration %s: %v", n, err)
		}
	}

	// Verify core tables exist: evidence, audit
	var cnt int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='evidence'`).Scan(&cnt)
	if err != nil {
		t.Fatalf("query evidence table: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("evidence table not found after migrations")
	}
	err = pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='audit'`).Scan(&cnt)
	if err != nil {
		t.Fatalf("query audit table: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("audit table not found after migrations")
	}
}
