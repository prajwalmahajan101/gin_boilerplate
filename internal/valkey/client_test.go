package valkey

import (
	"context"
	"testing"
	"time"
)

func TestNilClient_FailsOpen(t *testing.T) {
	var c *Client
	ctx := context.Background()

	if err := c.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if v, err := c.Get(ctx, "any"); v != "" || err != nil {
		t.Fatalf("get: v=%q err=%v", v, err)
	}
	if err := c.SetEX(ctx, "any", "val", time.Second); err != nil {
		t.Fatalf("setex: %v", err)
	}
	if ok, err := c.SetNX(ctx, "any", "val", time.Second); ok || err != nil {
		t.Fatalf("setnx: ok=%v err=%v", ok, err)
	}
	if n, err := c.Incr(ctx, "any"); n != 0 || err != nil {
		t.Fatalf("incr: n=%d err=%v", n, err)
	}
	if err := c.Expire(ctx, "any", time.Second); err != nil {
		t.Fatalf("expire: %v", err)
	}
	if err := c.Del(ctx, "any"); err != nil {
		t.Fatalf("del: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}
