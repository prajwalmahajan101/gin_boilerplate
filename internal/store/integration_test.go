//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

// Item is a test-local model mapped to the migrated items table. The real items
// module model lands in M5; this exercises the migration + Repository + Service.
type Item struct {
	store.BaseModel
	Name  string `db:"name"`
	Code  string `db:"code"`
	Notes []byte `db:"notes"`
}

func (Item) TableName() string { return "items" }

var dbCounter atomic.Int64

// newTestDB creates a throwaway database inside the Postgres server pointed to by
// DATABASE_URL, migrates it, and returns a pool. The database is dropped on cleanup.
func newTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	base := os.Getenv("DATABASE_URL")
	if base == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	name := fmt.Sprintf("gin_boilerplate_test_%d_%d", os.Getpid(), dbCounter.Add(1))

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

func TestRepository_CreateReadRoundTrip(t *testing.T) {
	pool := newTestDB(t)
	repo := store.NewRepository[Item](pool)
	ctx := context.Background()

	it := &Item{Name: "Widget", Code: "W-1", Notes: []byte(`{"seed":true}`)}
	if err := repo.Create(ctx, it); err != nil {
		t.Fatalf("create: %v", err)
	}
	if it.ID == 0 {
		t.Fatal("id not populated after create")
	}
	if !it.IsActive {
		t.Fatal("is_active not set true on create")
	}

	got, found, err := repo.GetByID(ctx, it.ID)
	if err != nil || !found {
		t.Fatalf("get by id: found=%v err=%v", found, err)
	}
	if got.Name != "Widget" || got.Code != "W-1" {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
	var notes map[string]any
	if err := json.Unmarshal(got.Notes, &notes); err != nil {
		t.Fatalf("notes not valid jsonb: %v", err)
	}
	if notes["seed"] != true {
		t.Fatalf("notes round-trip mismatch: %v", notes)
	}
}

func TestRepository_ListAndPaginate(t *testing.T) {
	pool := newTestDB(t)
	repo := store.NewRepository[Item](pool)
	ctx := context.Background()

	for i := range 3 {
		it := &Item{Name: fmt.Sprintf("n%d", i), Code: fmt.Sprintf("C-%d", i)}
		if err := repo.Create(ctx, it); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	items, total, err := repo.Paginate(ctx, 1, 10)
	if err != nil {
		t.Fatalf("paginate: %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("expected 3 items, got total=%d len=%d", total, len(items))
	}
}

func TestRepository_SoftDelete(t *testing.T) {
	pool := newTestDB(t)
	repo := store.NewRepository[Item](pool)
	ctx := context.Background()

	it := &Item{Name: "gone", Code: "G-1"}
	if err := repo.Create(ctx, it); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.SoftDelete(ctx, it.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	if _, active, _ := repo.GetActiveByID(ctx, it.ID); active {
		t.Fatal("soft-deleted row still active")
	}
	got, found, err := repo.GetByID(ctx, it.ID)
	if err != nil || !found {
		t.Fatalf("row should still exist: found=%v err=%v", found, err)
	}
	if got.IsActive {
		t.Fatal("is_active should be false after soft delete")
	}
}

// countingHooks records hook invocation order to prove BaseService fires them.
type countingHooks struct {
	store.NoOpHooks[Item]
	seq *[]string
}

func (h countingHooks) PreCreate(_ context.Context, _ *Item) error {
	*h.seq = append(*h.seq, "pre_create")
	return nil
}
func (h countingHooks) PostCreate(_ context.Context, _ *Item) { *h.seq = append(*h.seq, "post_create") }
func (h countingHooks) PreUpdate(_ context.Context, _ *Item) error {
	*h.seq = append(*h.seq, "pre_update")
	return nil
}
func (h countingHooks) PostUpdate(_ context.Context, _ *Item) { *h.seq = append(*h.seq, "post_update") }
func (h countingHooks) PreDelete(_ context.Context, _ *Item) error {
	*h.seq = append(*h.seq, "pre_delete")
	return nil
}
func (h countingHooks) PostDelete(_ context.Context, _ *Item) { *h.seq = append(*h.seq, "post_delete") }

func TestBaseService_HooksFireInOrder(t *testing.T) {
	pool := newTestDB(t)
	repo := store.NewRepository[Item](pool)
	seq := &[]string{}
	svc := store.NewBaseService[Item](repo, pool, countingHooks{seq: seq})
	ctx := context.Background()

	it := &Item{Name: "hooked", Code: "H-1"}
	if err := svc.Create(ctx, it); err != nil {
		t.Fatalf("service create: %v", err)
	}
	it.Name = "hooked2"
	if err := svc.Update(ctx, it); err != nil {
		t.Fatalf("service update: %v", err)
	}
	if err := svc.Delete(ctx, it.ID, true); err != nil {
		t.Fatalf("service delete: %v", err)
	}

	want := "pre_create,post_create,pre_update,post_update,pre_delete,post_delete"
	if got := strings.Join(*seq, ","); got != want {
		t.Fatalf("hook order = %q, want %q", got, want)
	}
}

func TestPool_PerCallQueryTimeout(t *testing.T) {
	pool := newTestDB(t) // DBQueryTimeoutMS=300 from newTestDB cfg
	ctx, cancel := store.WithQueryTimeout(context.Background())
	defer cancel()

	start := time.Now()
	_, err := pool.Exec(ctx, "SELECT pg_sleep(1)")
	if err == nil {
		t.Fatal("expected deadline error for slow query, got nil")
	}
	if elapsed := time.Since(start); elapsed > 900*time.Millisecond {
		t.Fatalf("query ran %v; timeout did not fire", elapsed)
	}
}
