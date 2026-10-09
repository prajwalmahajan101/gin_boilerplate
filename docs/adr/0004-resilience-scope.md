# 0004 — Resilience scope: in-process for v0.1.0

**Status:** Accepted · 2026-10 · v0.1.0

## Context

The service makes outbound calls (Valkey cache, and — as the boilerplate grows —
HTTP dependencies, S3, SES) and must stay healthy when a dependency degrades. The
full menu of resilience patterns spans in-process primitives (circuit breaker,
retry with backoff, timeouts, an SSRF guard) and distributed/coordinated ones
(shared breaker state across replicas, a service mesh, outbox/DLQ). Building all
of it before there is traffic is premature.

## Decision

Ship **in-process** resilience primitives for v0.1.0 and defer distributed /
two-tier coordination to **v0.2**:

- In scope now (`internal/platform/resilience`, `internal/platform/cache`):
  - Circuit breaker — hand-rolled, consecutive-failure, `CLOSED → OPEN →
    HALF_OPEN`, per-process. Deliberately in-memory: a Valkey-backed breaker is
    useless when Valkey is the thing that is down.
  - Retry with exponential backoff + full jitter; stops on non-retryable errors.
  - SSRF guard (`AssertPublicURL`) rejecting loopback/private/link-local hosts.
  - Composed `Policy` executor: rate-limit → retry → breaker → timeout.
  - Cache fail-open: a cache/breaker failure falls through to the origin, never
    errors the request.
- Deferred to v0.2:
  - Breaker/limiter state shared across replicas (e.g. coordinated via Valkey).
  - Transactional outbox + dead-letter queues for async event producers.
  - Service-mesh-level retries/timeouts.

## Consequences

- **+** Each replica protects itself immediately with no extra infrastructure.
- **+** Primitives are small, testable, and dependency-free.
- **−** Breaker and rate-limit decisions are per-process: N replicas can each send
  one trial request in `HALF_OPEN`, and rate limits are approximate under the
  memory fallback. Acceptable at v0.1.0 scale; the v0.2 distributed tier is the
  upgrade path.
- **−** No cross-service retry budget or DLQ yet; poison-message handling is a
  future milestone.

## Usage

- `internal/platform/resilience/{breaker,retry,ssrf,classify,executor}.go`.
- `internal/platform/cache/` — tiered cache with fail-open + TTL jitter.
- Rate limiting: `internal/platform/ratelimit` with a Valkey primary and
  in-memory fallback.
- When moving to v0.2, revisit this ADR and supersede it with the distributed
  design.
