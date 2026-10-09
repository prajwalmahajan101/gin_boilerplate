# 0003 — Typed errors with stable codes

**Status:** Accepted · 2026-10 · v0.1.0

## Context

Business and infrastructure code must signal failures in a way that handlers can
translate into the right HTTP status and a stable, client-facing error code —
without every handler re-deriving status codes from sentinel errors. The
resilience layer additionally needs to classify errors: which ones should trip a
circuit breaker, and which are worth retrying. Plain `errors.New` / `fmt.Errorf`
carry no status, code, or classification.

## Decision

Use a single typed error, `apperr.AppError`, with a registry of **stable string
codes**:

- Fields: `Code` (stable string, e.g. `VALIDATION_ERROR`, `NOT_FOUND`,
  `RATE_LIMITED`, `SERVICE_UNAVAILABLE`), `Message`, `HTTPStatus`, optional
  `Details`, `RequestID`, a wrapped `cause`, and a `trips` flag.
- Per-code constructors (`apperr.NotFound`, `apperr.ValidationError`,
  `apperr.TransientError`, …) set the matching HTTP status.
- `errors.As`-friendly: `AsAppError`, `Is(err, code)`, and `Unwrap` for chains.
- Classification helpers:
  - `TripsBreaker(err)` — true only for transient/timeout errors (the `trips`
    flag), so a breaker opens on real dependency failures, not validation errors.
  - `Retryable(err)` (resilience layer) — transient/timeout/external codes plus
    context-deadline and net timeouts; everything else fails safe.

## Consequences

- **+** Handlers render any error with `response.Error(c, err)`; the status and
  code come from the error itself.
- **+** Error codes are a stable part of the public contract — clients branch on
  `code`, not on message text or HTTP status alone.
- **+** The breaker and retry logic reuse the same error taxonomy instead of
  inventing their own.
- **−** Call sites must choose the right constructor; a bare `errors.New` returned
  from a service degrades to `INTERNAL_ERROR` (500) via `AsAppError` — safe, but
  loses specificity.
- **−** Adding a code is a deliberate change to the registry, not ad hoc.

## Usage

- `internal/platform/apperr/apperr.go` — type, code registry, constructors,
  `TripsBreaker`, `Is`, `AsAppError`.
- `internal/platform/resilience/classify.go` — `Retryable`, consumed by the retry
  loop and the composed executor.
- Services return `apperr.*` errors; handlers pass them to `response.Error`.
