package resilience

import (
	"context"
	"math/rand/v2"
	"time"
)

// RetryConfig tunes the backoff loop. A zero or negative MaxAttempts means "run
// once, no retry". Base is the first backoff interval; Cap bounds the longest
// single sleep so exponential growth cannot run away. Cap <= 0 means uncapped.
type RetryConfig struct {
	MaxAttempts int
	Base        time.Duration
	Cap         time.Duration
}

// Do runs fn, retrying on transient errors (see Retryable) with exponential
// backoff and full jitter. It returns nil on the first success, the error
// immediately if it is non-retryable, or the last error once attempts are
// exhausted. If ctx is cancelled during a backoff sleep, ctx.Err() is returned.
func Do(ctx context.Context, cfg RetryConfig, fn func(context.Context) error) error {
	attempts := cfg.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		err = fn(ctx)
		if err == nil || !Retryable(err) {
			return err
		}
		if attempt == attempts-1 {
			break // exhausted — do not sleep after the final attempt
		}
		if sleepErr := sleep(ctx, backoff(cfg, attempt)); sleepErr != nil {
			return sleepErr
		}
	}
	return err
}

// backoff returns the full-jitter interval for a zero-based attempt number:
// a uniform random value in [0, min(Cap, Base*2^attempt)] (AWS "full jitter").
// Jitter spreads retries from concurrent callers so they do not thunder together.
func backoff(cfg RetryConfig, attempt int) time.Duration {
	if cfg.Base <= 0 {
		return 0
	}
	ceiling := cfg.Base << attempt
	if ceiling <= 0 { // shift overflow on a huge attempt count
		ceiling = cfg.Cap
	}
	if cfg.Cap > 0 && ceiling > cfg.Cap {
		ceiling = cfg.Cap
	}
	if ceiling <= 0 {
		return 0
	}
	//nolint:gosec // backoff jitter does not need crypto-strength randomness
	return time.Duration(rand.Int64N(int64(ceiling) + 1))
}

// sleep waits for d or until ctx is done, whichever comes first.
func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
