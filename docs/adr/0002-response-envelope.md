# 0002 — Unified JSON response envelope

**Status:** Accepted · 2026-10 · v0.1.0

## Context

A REST API needs a predictable response shape so clients can parse success and
failure uniformly, correlate requests with logs, and surface structured error
details. Two broad options: return bare resource JSON (status code carries
success/failure) or wrap every response in a consistent envelope. This
boilerplate is one of a trio (Django / FastAPI / Gin) that share the same
contract, so the envelope must match its siblings.

## Decision

Wrap **every** response — success and error — in a single envelope:

```json
{
  "success": true,
  "message": "human-readable summary",
  "data": { },
  "errors": [ { "code": "...", "field": "...", "message": "..." } ],
  "request_id": "uuid"
}
```

- `success` is an explicit boolean, not inferred from the status code.
- `data` is present on success (omitted on error); `errors` is present on error.
- `request_id` is echoed on every response for end-to-end correlation.
- One constructor path builds success (`response.Success`) and error
  (`response.Error`) envelopes; handlers never hand-assemble JSON.

## Consequences

- **+** Clients parse one shape everywhere; errors always carry stable codes and
  optional field-level detail.
- **+** `request_id` in the body (and header) ties a client-visible response to
  server logs without digging.
- **+** Error rendering is centralized: `response.Error` maps a typed
  `apperr.AppError` to the right HTTP status + envelope (see ADR 0003).
- **−** Slightly more verbose than bare JSON; `data` nests resources one level
  deep. Accepted for consistency across the trio.
- **−** Pagination metadata rides inside the envelope rather than in headers.

## Usage

- `internal/platform/response/response.go` — `Envelope`, `Success`, `Error`.
- Handlers call `response.Success(c, http.StatusOK, "msg", data)` or
  `response.Error(c, err)`; the latter unwraps `*apperr.AppError`.
- `request_id` is injected by the request-ID middleware and read from the
  request context.
