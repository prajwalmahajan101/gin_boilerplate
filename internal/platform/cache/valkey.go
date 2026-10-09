package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type valkeyCache struct {
	rdb       *redis.Client
	prefix    string
	jitterPct int // ±pct spread on Set TTL to avoid synchronized expiry (T24)
}

func newValkeyCache(name string, rdb *redis.Client, jitterPct int) *valkeyCache {
	return &valkeyCache{rdb: rdb, prefix: "cache:" + name + ":", jitterPct: jitterPct}
}

func (c *valkeyCache) key(k string) string {
	return c.prefix + k
}

// Get returns the real error on a backend failure so a wrapping breaker can
// count it; a genuine miss (redis.Nil) is (nil, false, nil). Fail-open lives in
// the FailOpen decorator, not here.
func (c *valkeyCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	val, err := c.rdb.Get(ctx, c.key(key)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil // genuine miss
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (c *valkeyCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, c.key(key), val, jitteredTTL(ttl, c.jitterPct)).Err()
}

func (c *valkeyCache) Delete(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, c.key(key)).Err()
}
