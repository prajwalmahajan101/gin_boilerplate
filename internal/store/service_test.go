package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/pagination"
)

// hookTracker records which hooks were called and in what order.
type hookTracker[T Model] struct {
	NoOpHooks[T]
	calls  []string
	preErr error // if set, PreCreate/PreUpdate/PreDelete return this
}

func (h *hookTracker[T]) PreCreate(_ context.Context, _ *T) error {
	h.calls = append(h.calls, "PreCreate")
	return h.preErr
}

func (h *hookTracker[T]) PostCreate(_ context.Context, _ *T) {
	h.calls = append(h.calls, "PostCreate")
}

func (h *hookTracker[T]) PreUpdate(_ context.Context, _ *T) error {
	h.calls = append(h.calls, "PreUpdate")
	return h.preErr
}

func (h *hookTracker[T]) PostUpdate(_ context.Context, _ *T) {
	h.calls = append(h.calls, "PostUpdate")
}

func (h *hookTracker[T]) PreDelete(_ context.Context, _ *T) error {
	h.calls = append(h.calls, "PreDelete")
	return h.preErr
}

func (h *hookTracker[T]) PostDelete(_ context.Context, _ *T) {
	h.calls = append(h.calls, "PostDelete")
}

// --- Service tests need a pool, but we can't use a real pool in unit tests.
// We test the non-transactional methods (GetByID, GetByIDOrFail, List)
// and test hook ordering + cascade separately.

func TestGetByIDOrFail_NotFound(t *testing.T) {
	mq := &mockQuerier{rows: &mockRows{}}
	repo := NewRepository[testModel](mq)
	svc := NewBaseService[testModel](repo, nil, nil)

	_, err := svc.GetByIDOrFail(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
	if !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestGetActiveByIDOrFail_NotFound(t *testing.T) {
	mq := &mockQuerier{rows: &mockRows{}}
	repo := NewRepository[testModel](mq)
	svc := NewBaseService[testModel](repo, nil, nil)

	_, err := svc.GetActiveByIDOrFail(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
	if !apperr.Is(err, apperr.CodeNotFound) {
		t.Fatalf("expected NOT_FOUND, got %v", err)
	}
}

func TestList_ReturnsMeta(t *testing.T) {
	// Mock: List returns empty, Count returns 25
	mq := &mockQuerier{rows: &mockRows{}, row: &mockRow{id: 25}}
	repo := NewRepository[testModel](mq)
	svc := NewBaseService[testModel](repo, nil, nil)

	items, meta, err := svc.List(context.Background(), 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
	want := pagination.NewMeta(2, 10, 25)
	if meta != want {
		t.Errorf("meta: got %+v, want %+v", meta, want)
	}
}

func TestNoOpHooks_SatisfiesInterface(t *testing.T) {
	var h ServiceHooks[testModel] = &NoOpHooks[testModel]{}
	ctx := context.Background()
	m := &testModel{}
	if err := h.PreCreate(ctx, m); err != nil {
		t.Fatal(err)
	}
	h.PostCreate(ctx, m)
	if err := h.PreUpdate(ctx, m); err != nil {
		t.Fatal(err)
	}
	h.PostUpdate(ctx, m)
	if err := h.PreDelete(ctx, m); err != nil {
		t.Fatal(err)
	}
	h.PostDelete(ctx, m)
}

func TestHookTracker_RecordsCalls(t *testing.T) {
	h := &hookTracker[testModel]{}
	ctx := context.Background()
	m := &testModel{}

	_ = h.PreCreate(ctx, m)
	h.PostCreate(ctx, m)
	_ = h.PreUpdate(ctx, m)
	h.PostUpdate(ctx, m)

	want := []string{"PreCreate", "PostCreate", "PreUpdate", "PostUpdate"}
	if len(h.calls) != len(want) {
		t.Fatalf("calls: got %v, want %v", h.calls, want)
	}
	for i := range want {
		if h.calls[i] != want[i] {
			t.Errorf("call %d: got %q, want %q", i, h.calls[i], want[i])
		}
	}
}

func TestHookTracker_PreError_StopsChain(t *testing.T) {
	h := &hookTracker[testModel]{preErr: errors.New("validation failed")}
	ctx := context.Background()
	m := &testModel{}

	err := h.PreCreate(ctx, m)
	if err == nil {
		t.Fatal("expected error")
	}
	// PostCreate should NOT be called since PreCreate failed
	if len(h.calls) != 1 || h.calls[0] != "PreCreate" {
		t.Fatalf("calls: got %v, want [PreCreate]", h.calls)
	}
}

// --- Cascade tests ---

// treeCascade models a parent->children adjacency map. SoftDeleteByParent returns
// the children of the given id, so CascadeSoftDelete can walk the whole tree.
type treeCascade struct {
	children map[int64][]int64
	visited  []int64
	err      error
}

func (c *treeCascade) SoftDeleteByParent(_ context.Context, parentID int64) ([]int64, error) {
	c.visited = append(c.visited, parentID)
	if c.err != nil {
		return nil, c.err
	}
	return c.children[parentID], nil
}

func TestCascadeSoftDelete_CallsTargets(t *testing.T) {
	c1 := &treeCascade{}
	c2 := &treeCascade{}

	err := CascadeSoftDelete(context.Background(), 42, []CascadeTarget{c1, c2})
	if err != nil {
		t.Fatal(err)
	}
	if len(c1.visited) != 1 || c1.visited[0] != 42 {
		t.Errorf("c1 visited=%v", c1.visited)
	}
	if len(c2.visited) != 1 || c2.visited[0] != 42 {
		t.Errorf("c2 visited=%v", c2.visited)
	}
}

func TestCascadeSoftDelete_WalksTreeBreadthFirst(t *testing.T) {
	// 1 -> {2,3}; 2 -> {4}; 3 -> {5}; 4,5 leaves.
	c := &treeCascade{children: map[int64][]int64{
		1: {2, 3},
		2: {4},
		3: {5},
	}}
	if err := CascadeSoftDelete(context.Background(), 1, []CascadeTarget{c}); err != nil {
		t.Fatal(err)
	}
	// Every node, root included, must be visited exactly once.
	want := map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: true}
	if len(c.visited) != len(want) {
		t.Fatalf("visited %v, want all of %v once", c.visited, want)
	}
	seen := map[int64]int{}
	for _, id := range c.visited {
		seen[id]++
	}
	for id := range want {
		if seen[id] != 1 {
			t.Errorf("node %d visited %d times, want 1 (visited=%v)", id, seen[id], c.visited)
		}
	}
}

func TestCascadeSoftDelete_DepthCapStopsCycle(t *testing.T) {
	// Self-referential cycle: 1 -> 1 would loop forever without the depth cap.
	c := &treeCascade{children: map[int64][]int64{1: {1}}}
	done := make(chan error, 1)
	go func() { done <- CascadeSoftDelete(context.Background(), 1, []CascadeTarget{c}) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cascade did not terminate; depth cap failed")
	}
	// Bounded by the depth cap, not infinite.
	if len(c.visited) > MaxCascadeDepth+1 {
		t.Fatalf("visited %d nodes, want <= %d", len(c.visited), MaxCascadeDepth+1)
	}
}

func TestCascadeSoftDelete_TargetError(t *testing.T) {
	c := &treeCascade{err: errors.New("db fail")}
	err := CascadeSoftDelete(context.Background(), 1, []CascadeTarget{c})
	if err == nil {
		t.Fatal("expected error")
	}
}
