package reqcontext

import (
	"context"
	"sync/atomic"
	"time"
)

type timing struct {
	serviceNS atomic.Int64
	repoNS    atomic.Int64
}

type timingCtxKey struct{}

var timingKey = timingCtxKey{}

func WithTiming(ctx context.Context) context.Context {
	return context.WithValue(ctx, timingKey, &timing{})
}

func timingFrom(ctx context.Context) *timing {
	t, _ := ctx.Value(timingKey).(*timing)
	return t
}

func AddServiceTime(ctx context.Context, d time.Duration) {
	if t := timingFrom(ctx); t != nil {
		t.serviceNS.Add(int64(d))
	}
}

func AddRepoTime(ctx context.Context, d time.Duration) {
	if t := timingFrom(ctx); t != nil {
		t.repoNS.Add(int64(d))
	}
}

func TrackService(ctx context.Context) func() {
	start := time.Now()
	return func() { AddServiceTime(ctx, time.Since(start)) }
}

func TrackRepo(ctx context.Context) func() {
	start := time.Now()
	return func() { AddRepoTime(ctx, time.Since(start)) }
}

func TimingFromContext(ctx context.Context) (serviceMS, repoMS int64) {
	t := timingFrom(ctx)
	if t == nil {
		return 0, 0
	}
	return t.serviceNS.Load() / int64(time.Millisecond), t.repoNS.Load() / int64(time.Millisecond)
}
