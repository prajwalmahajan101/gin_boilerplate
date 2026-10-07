//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/dbtest"
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

func TestRepository_CreateReadRoundTrip(t *testing.T) {
	pool := dbtest.New(t)
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
	pool := dbtest.New(t)
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
	pool := dbtest.New(t)
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
	pool := dbtest.New(t)
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
	pool := dbtest.New(t) // dbtest sets DBQueryTimeoutMS=300
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
