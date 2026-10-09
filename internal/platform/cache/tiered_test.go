package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/resilience"
)

// flakyL2 stands in for Valkey: Get always fails, counting how many times it is
// actually invoked so a test can prove the breaker stops dialing a dead backend.
type flakyL2 struct{ getCalls int }

func (f *flakyL2) Get(context.Context, string) ([]byte, bool, error) {
	f.getCalls++
	return nil, false, errors.New("valkey down")
}
func (f *flakyL2) Set(context.Context, string, []byte, time.Duration) error { return nil }
func (f *flakyL2) Delete(context.Context, string) error                     { return nil }

func newTiered(l1, l2 Cache) *tieredCache { return &tieredCache{l1: l1, l2: l2} }

// TestTiered_L1AbsorbsOutage: a key already in L1 is served from L1 during an L2
// outage — the L2 backend is never touched (T22a: DB load stays flat).
func TestTiered_L1AbsorbsOutage(t *testing.T) {
	l2 := &flakyL2{}
	tc := newTiered(newLRUCache(100, time.Minute), newBreakerCache(l2, resilience.NewMemory("t", resilience.Config{FailThreshold: 5, Recovery: time.Minute})))
	ctx := context.Background()

	_ = tc.l1.Set(ctx, "k", []byte("v"), 0) // warm L1 only

	v, hit, err := tc.Get(ctx, "k")
	if err != nil || !hit || string(v) != "v" {
		t.Fatalf("got (%q,%v,%v), want (\"v\",true,nil)", v, hit, err)
	}
	if l2.getCalls != 0 {
		t.Fatalf("L2 dialed %d times, want 0 (L1 must absorb)", l2.getCalls)
	}
}

// TestTiered_BreakerSkipsDeadL2: for a cold key (not in L1) a down L2 is dialed
// only up to the breaker threshold, then OPEN short-circuits further dials
// (T22b: no per-request dial tax). Every Get still succeeds as a fail-open miss.
func TestTiered_BreakerSkipsDeadL2(t *testing.T) {
	l2 := &flakyL2{}
	const threshold = 3
	tc := newTiered(newLRUCache(100, time.Minute), newBreakerCache(l2, resilience.NewMemory("t", resilience.Config{FailThreshold: threshold, Recovery: time.Minute})))
	ctx := context.Background()

	for i := range 20 {
		if _, hit, err := tc.Get(ctx, "cold"); err != nil || hit {
			t.Fatalf("iter %d: got (hit %v, err %v), want fail-open miss", i, hit, err)
		}
	}
	if l2.getCalls != threshold {
		t.Fatalf("L2 dialed %d times, want %d (breaker must OPEN and stop dialing)", l2.getCalls, threshold)
	}
}
