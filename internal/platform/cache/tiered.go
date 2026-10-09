package cache

import (
	"context"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/resilience"
	"github.com/redis/go-redis/v9"
)

// tieredCache reads L1 (in-process) → L2 (breaker-guarded Valkey) → caller's DB
// (on a miss). An L2 hit populates L1 on the way back, so during a Valkey outage
// the working set is served from L1 and the DB read-rate stays flat instead of
// flooding (T22a, NFR-R5). Writes/deletes go to both tiers.
type tieredCache struct {
	l1 Cache
	l2 Cache
}

// TierConfig tunes the tiered cache. Zero L1 fields with L1Enabled=false yield a
// plain single-tier (fail-open) Valkey cache.
type TierConfig struct {
	L1Enabled            bool
	L1Max                int
	L1TTL                time.Duration
	BreakerFailThreshold int
	BreakerRecovery      time.Duration
	TTLJitterPct         int // ±pct spread on L2 Set TTL to avoid synchronized expiry (T24)
}

// NewTiered builds the hot-read cache for name. A nil rdb (Valkey disabled) falls
// back to a plain in-memory cache. With Valkey present it wraps it in a breaker;
// if L1Enabled it fronts that with a bounded LRU L1 tier.
func NewTiered(name string, rdb *redis.Client, cfg TierConfig) Cache {
	if rdb == nil {
		return newMemoryCache() // Valkey disabled — in-process fallback
	}
	l2 := newBreakerCache(
		newValkeyCache(name, rdb, cfg.TTLJitterPct),
		resilience.NewMemory("cache:"+name, resilience.Config{
			FailThreshold: cfg.BreakerFailThreshold,
			Recovery:      cfg.BreakerRecovery,
		}),
	)
	if !cfg.L1Enabled {
		return l2
	}
	return &tieredCache{l1: newLRUCache(cfg.L1Max, cfg.L1TTL), l2: l2}
}

func (c *tieredCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if v, hit, _ := c.l1.Get(ctx, key); hit {
		return v, true, nil
	}
	v, hit, _ := c.l2.Get(ctx, key)
	if hit {
		_ = c.l1.Set(ctx, key, v, 0) // L1 uses its own fixed TTL
	}
	return v, hit, nil
}

func (c *tieredCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	_ = c.l1.Set(ctx, key, val, ttl)
	return c.l2.Set(ctx, key, val, ttl)
}

func (c *tieredCache) Delete(ctx context.Context, key string) error {
	_ = c.l1.Delete(ctx, key)
	return c.l2.Delete(ctx, key)
}
