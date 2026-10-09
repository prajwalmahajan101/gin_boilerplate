package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

func TestDo_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Do(context.Background(), RetryConfig{MaxAttempts: 3}, func(context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("err=%v calls=%d, want nil/1", err, calls)
	}
}

func TestDo_StopsOnNonRetryable(t *testing.T) {
	calls := 0
	err := Do(context.Background(), RetryConfig{MaxAttempts: 5, Base: time.Nanosecond}, func(context.Context) error {
		calls++
		return apperr.ValidationError("bad input")
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 (non-retryable must not retry)", calls)
	}
	if !apperr.Is(err, apperr.CodeValidationError) {
		t.Fatalf("err=%v, want validation error", err)
	}
}

func TestDo_RetriesTransientThenSucceeds(t *testing.T) {
	calls := 0
	err := Do(context.Background(), RetryConfig{MaxAttempts: 4, Base: time.Nanosecond, Cap: time.Microsecond}, func(context.Context) error {
		calls++
		if calls < 3 {
			return apperr.TransientError("flaky")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("err=%v calls=%d, want nil/3", err, calls)
	}
}

func TestDo_ExhaustsAttempts(t *testing.T) {
	calls := 0
	err := Do(context.Background(), RetryConfig{MaxAttempts: 3, Base: time.Nanosecond}, func(context.Context) error {
		calls++
		return apperr.TransientError("always fails")
	})
	if calls != 3 {
		t.Fatalf("calls=%d, want 3", calls)
	}
	if !apperr.Is(err, apperr.CodeTransientError) {
		t.Fatalf("err=%v, want last transient error", err)
	}
}

func TestDo_ContextCancelAbortsBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Do(ctx, RetryConfig{MaxAttempts: 5, Base: time.Hour}, func(context.Context) error {
		calls++
		cancel() // cancel before the (1h) backoff sleep
		return apperr.TransientError("retryable")
	})
	if calls != 1 {
		t.Fatalf("calls=%d, want 1 (cancel must abort before 2nd attempt)", calls)
	}
	if err != context.Canceled {
		t.Fatalf("err=%v, want context.Canceled", err)
	}
}

// backoff must never exceed Cap and must be >= 0.
func TestBackoff_BoundedByCap(t *testing.T) {
	cfg := RetryConfig{Base: 10 * time.Millisecond, Cap: 50 * time.Millisecond}
	for attempt := 0; attempt < 20; attempt++ {
		d := backoff(cfg, attempt)
		if d < 0 || d > cfg.Cap {
			t.Fatalf("attempt %d: backoff=%v out of [0,%v]", attempt, d, cfg.Cap)
		}
	}
}

func TestBackoff_ZeroBase(t *testing.T) {
	if d := backoff(RetryConfig{Base: 0, Cap: time.Second}, 3); d != 0 {
		t.Fatalf("zero base must give zero backoff, got %v", d)
	}
}
