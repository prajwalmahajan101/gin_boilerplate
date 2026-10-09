// Package resilience holds the outbound-safety primitives. breaker is a circuit
// breaker over a dependency: CLOSED passes calls and counts breaker-tripping
// failures; at the threshold it trips OPEN and rejects fast with a
// ServiceUnavailable error without calling fn; after the recovery window it
// admits a trial (HALF_OPEN) that either closes or re-opens it (NFR-R1/R5).
//
// Hand-rolled, consecutive-count model (no breaker library). Only the in-memory
// implementation is provided — the cache breaker must be in-process, since a
// Valkey-backed breaker is useless when Valkey is the thing that is down.
package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "closed"
	}
}

// Breaker guards a named dependency.
type Breaker interface {
	// Call runs fn unless the breaker is OPEN, in which case it returns a
	// ServiceUnavailable error without calling fn. Only breaker-tripping errors
	// count toward opening.
	Call(ctx context.Context, fn func() error) error
	State() State
}

// Config tunes a breaker.
type Config struct {
	FailThreshold int
	Recovery      time.Duration
}

// NewMemory returns an in-process breaker for name.
func NewMemory(name string, cfg Config) Breaker {
	return newMemoryBreaker(name, cfg)
}

type memoryBreaker struct {
	name string
	cfg  Config

	mu       sync.Mutex
	failures int
	state    State
	openedAt time.Time
}

func newMemoryBreaker(name string, cfg Config) *memoryBreaker {
	return &memoryBreaker{name: name, cfg: cfg}
}

// stateLocked returns the current state, transitioning OPEN→HALF_OPEN once the
// recovery window has elapsed. Caller holds mu.
func (b *memoryBreaker) stateLocked() State {
	if b.state == StateOpen && time.Since(b.openedAt) >= b.cfg.Recovery {
		b.state = StateHalfOpen
	}
	return b.state
}

func (b *memoryBreaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateLocked()
}

// Allow reports whether a call may proceed (CLOSED or a HALF_OPEN trial).
func (b *memoryBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateLocked() != StateOpen
}

// Record feeds a call outcome into the breaker: success closes it, a
// breaker-tripping failure increments toward (or trips) OPEN, and any other
// error is ignored.
func (b *memoryBreaker) Record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	st := b.stateLocked()
	switch {
	case err == nil:
		b.failures = 0
		b.state = StateClosed
	case apperr.TripsBreaker(err):
		b.failures++
		// A failed trial in HALF_OPEN, or hitting the threshold in CLOSED, opens.
		if st == StateHalfOpen || b.failures >= b.cfg.FailThreshold {
			b.state = StateOpen
			b.openedAt = time.Now()
		}
	}
}

func (b *memoryBreaker) Call(_ context.Context, fn func() error) error {
	if !b.Allow() {
		return apperr.ServiceUnavailable(b.name)
	}
	err := fn()
	b.Record(err)
	return err
}
