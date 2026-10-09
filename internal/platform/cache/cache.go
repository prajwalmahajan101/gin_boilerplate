// Package cache is a small key/value cache with two interchangeable backends:
// an in-memory map (TTL) and Valkey. Every Valkey error is fail-open — logged
// and treated as a miss/no-op — so a cache outage never surfaces as a 5xx
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is the backend-agnostic surface. Get returns (value, hit, err); a
// fail-open backend reports a miss instead of an error.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Provider builds and caches Cache instances by name. Non-nil rdb → Valkey;
// nil → in-memory. The name namespaces keys so two logical caches never collide
// on a shared Valkey.
type Provider struct {
	rdb    *redis.Client
	mu     sync.Mutex
	caches map[string]Cache
}

func NewProvider(rdb *redis.Client) *Provider {
	return &Provider{rdb: rdb, caches: map[string]Cache{}}
}

func (p *Provider) Get(name string) Cache {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.caches[name]; ok {
		return c
	}
	var c Cache
	if p.rdb != nil {
		c = FailOpen(newValkeyCache(name, p.rdb, 0)) // valkey propagates errors; fail open here (no TTL jitter on this generic path)
	} else {
		c = newMemoryCache()
	}
	p.caches[name] = c
	return c
}
