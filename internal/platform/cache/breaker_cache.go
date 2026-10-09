package cache

import (
	"context"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/resilience"
)

// breakerCache guards an inner (Valkey) cache with a circuit breaker so a down or
// slow backend is skipped rather than dialed every request (T22b, NFR-R5). A
// backend error is mapped to a breaker-tripping TransientError; once the breaker
// OPENs, Call returns without invoking the inner op — no dial. Either way the
// outcome degrades to a miss / no-op, so the cache stays fail-open. HALF_OPEN
// probes re-close it when the backend returns.
type breakerCache struct {
	inner Cache
	br    resilience.Breaker
}

func newBreakerCache(inner Cache, br resilience.Breaker) *breakerCache {
	return &breakerCache{inner: inner, br: br}
}

func (c *breakerCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	var val []byte
	var hit bool
	err := c.br.Call(ctx, func() error {
		v, h, e := c.inner.Get(ctx, key)
		if e != nil {
			return apperr.TransientError("cache get: " + e.Error())
		}
		val, hit = v, h
		return nil
	})
	if err != nil {
		return nil, false, nil // breaker open (no dial) or transient failure → miss
	}
	return val, hit, nil
}

func (c *breakerCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	_ = c.br.Call(ctx, func() error {
		if e := c.inner.Set(ctx, key, val, ttl); e != nil {
			return apperr.TransientError("cache set: " + e.Error())
		}
		return nil
	})
	return nil
}

func (c *breakerCache) Delete(ctx context.Context, key string) error {
	_ = c.br.Call(ctx, func() error {
		if e := c.inner.Delete(ctx, key); e != nil {
			return apperr.TransientError("cache delete: " + e.Error())
		}
		return nil
	})
	return nil
}
