package cache

import (
	"math/rand/v2"
	"time"
)

// jitteredTTL spreads a base TTL by ±pct percent so that keys written in a burst
// do not all expire in the same instant (cache avalanche → DB thundering herd,
// T23/T24). Returns base unchanged when there is nothing to jitter: base <= 0
// (a zero ttl means "no expiry" in redis — jitter must never turn that into one)
// or pct <= 0, and it floors at base if a draw would make the result non-positive.
//
// ponytail: ±pct jitter only; full-jitter / decorrelated jitter not needed for a
// single fixed TTL — upgrade only if a multi-tier TTL scheme later needs it.
func jitteredTTL(base time.Duration, pct int) time.Duration {
	if base <= 0 || pct <= 0 {
		return base
	}
	// Two-sided: delta in [-pct%, +pct%] of base, so the mean stays at base.
	delta := float64(base) * float64(pct) / 100.0 * (2*rand.Float64() - 1) //nolint:gosec // TTL jitter does not need crypto-strength randomness
	out := base + time.Duration(delta)
	if out <= 0 {
		return base
	}
	return out
}
