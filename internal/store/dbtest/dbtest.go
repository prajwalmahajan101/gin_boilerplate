// Package dbtest provides a throwaway-database helper for integration tests.
// Each call creates a fresh database inside the Postgres server pointed to by
// DATABASE_URL, migrates it, and drops it on test cleanup — so tests never touch
// existing data and never collide.
package dbtest

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

var counter atomic.Int64

// New creates a migrated throwaway database and returns a pool to it. The test is
// skipped when DATABASE_URL is unset. The database is dropped on t.Cleanup.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()

	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	name := fmt.Sprintf("gin_boilerplate_test_%d_%d", os.Getpid(), counter.Add(1))

	admin, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatalf("connect admin: %v", err)
	}
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("create database %s: %v", name, err)
	}
	_ = admin.Close(ctx)

	testURL := swapDBName(t, base, name)
	if err = store.Migrate(testURL); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{DatabaseURL: testURL, DBConnTimeoutMS: 5000, DBQueryTimeoutMS: 300}
	pool, err := store.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		admin, err := pgx.Connect(ctx, base)
		if err != nil {
			t.Logf("cleanup connect: %v", err)
			return
		}
		defer admin.Close(ctx)
		if _, err := admin.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)"); err != nil {
			t.Logf("drop database %s: %v", name, err)
		}
	})
	return pool
}

func swapDBName(t *testing.T, raw, name string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}
