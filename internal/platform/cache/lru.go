package cache

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// lruCache is the bounded in-process L1 tier: an LRU capped at max entries with a
// short per-entry TTL. Bounded by construction so a Valkey outage (which pushes
// the whole working set through L1) cannot OOM the process (NFR-R5, CACHE_L1_MAX).
//
// Its own TTL is ignored — the TTL is fixed at construction from CACHE_L1_TTL_S;
// the ttl arg on Set exists only to satisfy the Cache interface.
type lruCache struct {
	lru *expirable.LRU[string, []byte]
}

func newLRUCache(maxEntries int, ttl time.Duration) *lruCache {
	return &lruCache{lru: expirable.NewLRU[string, []byte](maxEntries, nil, ttl)}
}

func (c *lruCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.lru.Get(key)
	return v, ok, nil
}

func (c *lruCache) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	c.lru.Add(key, val)
	return nil
}

func (c *lruCache) Delete(_ context.Context, key string) error {
	c.lru.Remove(key)
	return nil
}
