package resilience

import (
	"context"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/ratelimit"
)

// Policy composes the outbound-safety primitives into a single guarded call:
// rate-limit -> retry( breaker( timeout( fn ) ) ). Each field is optional; a
// nil Limiter / nil Breaker / zero Timeout simply skips that layer, so Policy{}
// degrades to "just run fn once".
type Policy struct {
	// Limiter gates the whole call before any attempt. Nil skips rate-limiting.
	Limiter ratelimit.Limiter
	RateKey string
	RPM     int

	// Breaker guards each attempt; an OPEN breaker short-circuits fast. Nil skips it.
	Breaker Breaker

	// Timeout bounds each individual attempt. Zero means no per-attempt timeout.
	Timeout time.Duration

	// Retry controls the backoff loop around breaker+timeout+fn.
	Retry RetryConfig
}

// Execute runs fn under the policy. Composition order follows the roadmap:
// rate-limit (once, outermost) -> retry -> breaker -> timeout -> fn.
func (p Policy) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Rate-limit once, before spending any attempt. A denied call never runs fn.
	if p.Limiter != nil {
		ok, err := p.Limiter.Allow(ctx, p.RateKey, p.RPM)
		if err != nil {
			return err
		}
		if !ok {
			return apperr.RateLimited("rate limit exceeded")
		}
	}

	return Do(ctx, p.Retry, func(ctx context.Context) error {
		return p.attempt(ctx, fn)
	})
}

// attempt runs a single try through the breaker and per-attempt timeout.
func (p Policy) attempt(ctx context.Context, fn func(context.Context) error) error {
	run := func() error {
		if p.Timeout <= 0 {
			return fn(ctx)
		}
		attemptCtx, cancel := context.WithTimeout(ctx, p.Timeout)
		defer cancel()
		return fn(attemptCtx)
	}
	if p.Breaker == nil {
		return run()
	}
	return p.Breaker.Call(ctx, run)
}
