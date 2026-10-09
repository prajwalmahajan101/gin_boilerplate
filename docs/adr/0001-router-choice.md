# 0001 — HTTP router: Gin

**Status:** Accepted · 2026-10 · v0.1.0

## Context

The service needs an HTTP router for a REST API with middleware (auth, rate
limiting, request-ID, recovery, body limits), path parameters, JSON binding +
validation, and grouped route trees (public / protected / admin). Candidates
considered: the standard library `net/http` + `ServeMux`, `chi`, `echo`, and
`gin`. This is a boilerplate meant to be forked, so ecosystem familiarity and
batteries-included ergonomics weigh as heavily as raw performance.

## Decision

Use **Gin**.

- Mature, widely adopted — the most recognizable choice for a Go REST starter,
  so forkers already know it.
- Built-in request binding + `validator`-tag validation (`binding:"required,email"`),
  which the handlers rely on for input validation at the edge.
- `RouterGroup` cleanly models the three auth gates (public / protected / admin)
  with per-group middleware.
- First-class Swagger integration via `swaggo/gin-swagger`.
- Large middleware ecosystem; trivial to write our own `gin.HandlerFunc`.

## Consequences

- **+** Fast to build on; conventions are well known; minimal glue for binding,
  validation, and docs.
- **+** Route grouping maps directly onto the public/protected/admin model.
- **−** A framework dependency rather than the stdlib; Gin owns the context type
  (`*gin.Context`), so handlers are coupled to it. Mitigated by keeping business
  logic in the service layer, which takes `context.Context`, not `*gin.Context`.
- **−** Gin's default `Logger`/`Recovery` are replaced with our own middleware
  for structured logging and the typed-error envelope.

## Usage

- Router assembly and the three gates live in
  `internal/platform/httpserver/router.go`.
- Each module exposes `RegisterRoutes(public, protected, admin *gin.RouterGroup)`.
- Handlers bind with `c.ShouldBindJSON(&req)` and translate errors through the
  response envelope; they never embed persistence or business logic.
