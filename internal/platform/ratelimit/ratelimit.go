package ratelimit

import "context"

type Limiter interface {
	Allow(ctx context.Context, key string, rpm int) (bool, error)
}

type fallback struct {
	primary, secondary Limiter
}

func WithFallback(primary, secondary Limiter) Limiter {
	return &fallback{primary: primary, secondary: secondary}
}

func (f *fallback) Allow(ctx context.Context, key string, rpm int) (bool, error) {
	ok, err := f.primary.Allow(ctx, key, rpm)
	if err != nil {
		return f.secondary.Allow(ctx, key, rpm)
	}
	return ok, nil
}
