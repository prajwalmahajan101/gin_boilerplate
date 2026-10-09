package resilience

import (
	"context"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

func newTestBreaker(recovery time.Duration) *memoryBreaker {
	return newMemoryBreaker("test", Config{FailThreshold: 3, Recovery: recovery})
}

func TestMemoryBreaker_TripsAtThreshold(t *testing.T) {
	b := newTestBreaker(time.Minute)
	tripping := apperr.TransientError("boom")

	for i := 0; i < 2; i++ {
		b.Record(tripping)
	}
	if b.State() != StateClosed {
		t.Fatalf("below threshold: state = %v, want closed", b.State())
	}
	b.Record(tripping) // 3rd → trips
	if b.State() != StateOpen {
		t.Fatalf("at threshold: state = %v, want open", b.State())
	}
}

func TestMemoryBreaker_OpenRejectsWithoutCallingFn(t *testing.T) {
	b := newTestBreaker(time.Minute)
	for i := 0; i < 3; i++ {
		b.Record(apperr.TransientError("boom"))
	}

	called := false
	err := b.Call(context.Background(), func() error {
		called = true
		return nil
	})
	if called {
		t.Fatal("fn called while breaker open")
	}
	if err == nil {
		t.Fatal("expected ServiceUnavailable, got nil")
	}
}

func TestMemoryBreaker_NonTrippingErrorIgnored(t *testing.T) {
	b := newTestBreaker(time.Minute)
	for i := 0; i < 5; i++ {
		b.Record(apperr.ValidationError("bad input")) // never trips
	}
	if b.State() != StateClosed {
		t.Fatalf("state = %v, want closed", b.State())
	}
}

func TestMemoryBreaker_RecoveryHalfOpenThenClose(t *testing.T) {
	b := newTestBreaker(10 * time.Millisecond)
	for i := 0; i < 3; i++ {
		b.Record(apperr.TransientError("boom"))
	}
	if b.State() != StateOpen {
		t.Fatalf("state = %v, want open", b.State())
	}

	time.Sleep(15 * time.Millisecond) // recovery elapses
	if b.State() != StateHalfOpen {
		t.Fatalf("after recovery: state = %v, want half_open", b.State())
	}
	if !b.Allow() {
		t.Fatal("half-open should admit a trial")
	}
	b.Record(nil) // trial succeeds → close
	if b.State() != StateClosed {
		t.Fatalf("after successful trial: state = %v, want closed", b.State())
	}
}

func TestMemoryBreaker_HalfOpenFailureReopens(t *testing.T) {
	b := newTestBreaker(10 * time.Millisecond)
	for i := 0; i < 3; i++ {
		b.Record(apperr.TransientError("boom"))
	}
	time.Sleep(15 * time.Millisecond)
	if b.State() != StateHalfOpen {
		t.Fatalf("state = %v, want half_open", b.State())
	}

	b.Record(apperr.TransientError("boom again")) // trial fails → re-open
	if b.State() != StateOpen {
		t.Fatalf("after failed trial: state = %v, want open", b.State())
	}
}
