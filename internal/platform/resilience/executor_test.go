package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

type fakeLimiter struct {
	allow bool
	err   error
	calls int
}

func (f *fakeLimiter) Allow(context.Context, string, int) (bool, error) {
	f.calls++
	return f.allow, f.err
}

func TestExecute_RateLimitDenied(t *testing.T) {
	lim := &fakeLimiter{allow: false}
	ran := false
	err := Policy{Limiter: lim, RPM: 1}.Execute(context.Background(), func(context.Context) error {
		ran = true
		return nil
	})
	if ran {
		t.Fatal("fn must not run when rate-limited")
	}
	if !apperr.Is(err, apperr.CodeRateLimited) {
		t.Fatalf("err=%v, want rate limited", err)
	}
}

func TestExecute_OpenBreakerShortCircuits(t *testing.T) {
	b := NewMemory("dep", Config{FailThreshold: 1, Recovery: time.Hour})
	// Trip it OPEN with one breaker-tripping failure.
	_ = b.Call(context.Background(), func() error { return apperr.TransientError("down") })
	if b.State() != StateOpen {
		t.Fatalf("precondition: breaker state=%v, want open", b.State())
	}

	ran := false
	err := Policy{Breaker: b, Retry: RetryConfig{MaxAttempts: 3, Base: time.Nanosecond}}.
		Execute(context.Background(), func(context.Context) error {
			ran = true
			return nil
		})
	if ran {
		t.Fatal("fn must not run while breaker is OPEN")
	}
	// ServiceUnavailable is non-retryable, so the loop stops immediately.
	if !apperr.Is(err, apperr.CodeServiceUnavail) {
		t.Fatalf("err=%v, want service unavailable", err)
	}
}

func TestExecute_RetriesTransientThenSucceeds(t *testing.T) {
	lim := &fakeLimiter{allow: true}
	calls := 0
	err := Policy{
		Limiter: lim,
		Retry:   RetryConfig{MaxAttempts: 4, Base: time.Nanosecond},
	}.Execute(context.Background(), func(context.Context) error {
		calls++
		if calls < 2 {
			return apperr.TransientError("flaky")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("err=%v calls=%d, want nil/2", err, calls)
	}
	if lim.calls != 1 {
		t.Fatalf("limiter called %d times, want 1 (gate once, not per retry)", lim.calls)
	}
}

func TestExecute_PerAttemptTimeout(t *testing.T) {
	var gotDeadline bool
	err := Policy{
		Timeout: 10 * time.Millisecond,
		Retry:   RetryConfig{MaxAttempts: 1},
	}.Execute(context.Background(), func(ctx context.Context) error {
		_, gotDeadline = ctx.Deadline()
		return nil
	})
	if err != nil {
		t.Fatalf("err=%v, want nil", err)
	}
	if !gotDeadline {
		t.Fatal("per-attempt timeout must set a deadline on fn's context")
	}
}

func TestExecute_BarePolicyRunsOnce(t *testing.T) {
	calls := 0
	err := Policy{}.Execute(context.Background(), func(context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want nil/1", err, calls)
	}
}
