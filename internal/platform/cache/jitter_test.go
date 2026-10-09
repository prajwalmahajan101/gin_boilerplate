package cache

import (
	"testing"
	"time"
)

func TestJitteredTTL_WithinBand(t *testing.T) {
	const base = 20 * time.Second
	const pct = 10
	lo := base - base*pct/100
	hi := base + base*pct/100

	seen := make(map[time.Duration]struct{})
	for range 1000 {
		got := jitteredTTL(base, pct)
		if got < lo || got > hi {
			t.Fatalf("jitteredTTL=%v out of band [%v,%v]", got, lo, hi)
		}
		seen[got] = struct{}{}
	}
	// Must actually vary — a constant TTL is the bug we're fixing.
	if len(seen) < 2 {
		t.Fatalf("expected varied TTLs, got %d distinct values", len(seen))
	}
}

func TestJitteredTTL_NoOpCases(t *testing.T) {
	const base = 20 * time.Second
	if got := jitteredTTL(base, 0); got != base {
		t.Fatalf("pct=0: got %v, want %v (unchanged)", got, base)
	}
	if got := jitteredTTL(0, 10); got != 0 {
		t.Fatalf("base=0 (no expiry): got %v, want 0 (must not become an expiry)", got)
	}
}
