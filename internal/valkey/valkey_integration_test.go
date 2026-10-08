//go:build integration

package valkey_test

import (
	"context"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

func newTestClient(t *testing.T) *valkey.Client {
	t.Helper()
	cfg := &config.Config{ValkeyURL: testURL(t)}
	c, err := valkey.New(cfg)
	if err != nil {
		t.Fatalf("valkey.New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func testURL(t *testing.T) string {
	t.Helper()
	url := "redis://localhost:6379/0"
	cfg := &config.Config{ValkeyURL: url}
	c, err := valkey.New(cfg)
	if err != nil {
		t.Skipf("VALKEY not reachable at %s: %v", url, err)
	}
	_ = c.Close()
	return url
}

func TestClient_PingSetGetDel(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	key := "test:valkey:setgetdel"
	if err := c.SetEX(ctx, key, "hello", 10*time.Second); err != nil {
		t.Fatalf("setex: %v", err)
	}
	got, err := c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
	if delErr := c.Del(ctx, key); delErr != nil {
		t.Fatalf("del: %v", delErr)
	}
	got, err = c.Get(ctx, key)
	if err != nil {
		t.Fatalf("get after del: %v", err)
	}
	if got != "" {
		t.Fatalf("got %q after del, want empty", got)
	}
}

func TestClient_IncrExpire(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := "test:valkey:incr"
	defer func() { _ = c.Del(ctx, key) }()

	n, err := c.Incr(ctx, key)
	if err != nil {
		t.Fatalf("incr: %v", err)
	}
	if n != 1 {
		t.Fatalf("incr = %d, want 1", n)
	}
	n, _ = c.Incr(ctx, key)
	if n != 2 {
		t.Fatalf("incr = %d, want 2", n)
	}
	if err := c.Expire(ctx, key, 5*time.Second); err != nil {
		t.Fatalf("expire: %v", err)
	}
}

func TestClient_SetNX(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()
	key := "test:valkey:setnx"
	defer func() { _ = c.Del(ctx, key) }()

	ok, err := c.SetNX(ctx, key, "first", 10*time.Second)
	if err != nil {
		t.Fatalf("setnx: %v", err)
	}
	if !ok {
		t.Fatal("setnx first call should return true")
	}
	ok, _ = c.SetNX(ctx, key, "second", 10*time.Second)
	if ok {
		t.Fatal("setnx second call should return false")
	}
}
