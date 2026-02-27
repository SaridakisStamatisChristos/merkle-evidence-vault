package integration

import (
	"context"
	"fmt"
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

	migDir, err := findMigrationsDir()
	if err != nil {
		t.Skipf("persistence migrations dir not found; skipping migration test: %v", err)
	}
	entries, err := os.ReadDir(migDir)
	if err != nil {
		t.Fatalf("read migrations dir %s: %v", migDir, err)
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

// findMigrationsDir looks for persistence/migrations by searching upward
// from the current working directory. Returns an error if not found.
func findMigrationsDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(wd, "persistence", "migrations")
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			return cand, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf("persistence/migrations not found in cwd or parents")
}
