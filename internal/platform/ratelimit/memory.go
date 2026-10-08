package ratelimit

import (
	"context"
	"sync"
	"time"
)

type window struct {
	count  int64
	expiry time.Time
}

type MemoryLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
}

func NewMemoryLimiter() *MemoryLimiter {
	return &MemoryLimiter{windows: make(map[string]*window)}
}

func (m *MemoryLimiter) Allow(_ context.Context, key string, rpm int) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	w, ok := m.windows[key]
	if !ok || now.After(w.expiry) {
		m.windows[key] = &window{count: 1, expiry: now.Add(time.Minute)}
		return true, nil
	}
	w.count++
	return w.count <= int64(rpm), nil
}
