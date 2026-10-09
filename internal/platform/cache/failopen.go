package cache

import (
	"context"
	"log/slog"
	"time"
)

// failOpenCache wraps a Cache so every backend error degrades to a miss / no-op
// instead of surfacing: a cache outage never becomes a 5xx (F-6). Used for the
// plain (non-tiered) Valkey path; the tiered path fails open at the breaker tier.
type failOpenCache struct{ inner Cache }

// FailOpen returns c wrapped so backend errors never propagate.
func FailOpen(c Cache) Cache { return &failOpenCache{inner: c} }

func (c *failOpenCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, hit, err := c.inner.Get(ctx, key)
	if err != nil {
		slog.Warn("cache: get failed, failing open", "key", key, "err", err)
		return nil, false, nil
	}
	return val, hit, nil
}

func (c *failOpenCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if err := c.inner.Set(ctx, key, val, ttl); err != nil {
		slog.Warn("cache: set failed, ignoring", "key", key, "err", err)
	}
	return nil
}

func (c *failOpenCache) Delete(ctx context.Context, key string) error {
	if err := c.inner.Delete(ctx, key); err != nil {
		slog.Warn("cache: delete failed, ignoring", "key", key, "err", err)
	}
	return nil
}
