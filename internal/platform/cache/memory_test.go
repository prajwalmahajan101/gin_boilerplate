package cache

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGetDelete(t *testing.T) {
	c := newMemoryCache()
	ctx := context.Background()

	if _, hit, _ := c.Get(ctx, "missing"); hit {
		t.Fatal("expected miss on empty cache")
	}

	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatalf("set: %v", err)
	}
	val, hit, _ := c.Get(ctx, "k")
	if !hit || string(val) != "v" {
		t.Fatalf("get = (%q, %v), want (\"v\", true)", val, hit)
	}

	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, hit, _ := c.Get(ctx, "k"); hit {
		t.Fatal("expected miss after delete")
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	c := newMemoryCache()
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), 10*time.Millisecond); err != nil {
		t.Fatalf("set: %v", err)
	}
	if _, hit, _ := c.Get(ctx, "k"); !hit {
		t.Fatal("expected hit before expiry")
	}

	time.Sleep(15 * time.Millisecond)
	if _, hit, _ := c.Get(ctx, "k"); hit {
		t.Fatal("expected miss after TTL expiry")
	}
}
