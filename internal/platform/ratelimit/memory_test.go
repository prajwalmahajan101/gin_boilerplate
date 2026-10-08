package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLimiter_Basic(t *testing.T) {
	m := NewMemoryLimiter()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ok, err := m.Allow(ctx, "k", 5)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	ok, err := m.Allow(ctx, "k", 5)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("6th request should be denied")
	}
}

func TestMemoryLimiter_WindowExpiry(t *testing.T) {
	m := NewMemoryLimiter()
	ctx := context.Background()

	m.Allow(ctx, "k", 1)

	ok, _ := m.Allow(ctx, "k", 1)
	if ok {
		t.Fatal("should be denied")
	}

	m.mu.Lock()
	m.windows["k"].expiry = time.Now().Add(-time.Second)
	m.mu.Unlock()

	ok, _ = m.Allow(ctx, "k", 1)
	if !ok {
		t.Fatal("should be allowed after expiry")
	}
}

func TestMemoryLimiter_SeparateKeys(t *testing.T) {
	m := NewMemoryLimiter()
	ctx := context.Background()

	m.Allow(ctx, "a", 1)

	ok, _ := m.Allow(ctx, "a", 1)
	if ok {
		t.Fatal("key a should be exhausted")
	}

	ok, _ = m.Allow(ctx, "b", 1)
	if !ok {
		t.Fatal("key b should be allowed")
	}
}

func TestWithFallback(t *testing.T) {
	mem := NewMemoryLimiter()
	failing := &failingLimiter{}
	fb := WithFallback(failing, mem)
	ctx := context.Background()

	ok, err := fb.Allow(ctx, "k", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("fallback should allow")
	}
}

type failingLimiter struct{}

func (f *failingLimiter) Allow(_ context.Context, _ string, _ int) (bool, error) {
	return false, context.DeadlineExceeded
}
