package cache

import (
	"context"
	"sync"
	"time"
)

type entry struct {
	val []byte
	exp time.Time // zero = no expiry
}

type memoryCache struct {
	m sync.Map // key string -> entry
}

func newMemoryCache() *memoryCache { return &memoryCache{} }

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false, nil
	}
	e := v.(entry)
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		c.m.Delete(key) // lazy expiry
		return nil, false, nil
	}
	return e.val, true, nil
}

func (c *memoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	c.m.Store(key, entry{val: val, exp: exp})
	return nil
}

func (c *memoryCache) Delete(_ context.Context, key string) error {
	c.m.Delete(key)
	return nil
}
