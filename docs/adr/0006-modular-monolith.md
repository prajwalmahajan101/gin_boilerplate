# 0006 — Modular monolith architecture

**Status:** Accepted · 2026-10 · v0.1.0

## Context

The service must be easy to reason about and fork while leaving room to grow. The
choice is between a layered-but-undivided monolith, a set of microservices, and a
**modular monolith** (one deployable, internally partitioned into modules with
explicit boundaries). Microservices add operational cost (network, deployment,
distributed data) unjustified at this stage; a flat monolith blurs boundaries and
rots. The boilerplate wants clear module seams now, with the option to extract a
module into its own service later.

## Decision

Build a **modular monolith**: one binary, partitioned into domain modules under
`internal/modules/` (e.g. `items`, `auth`), each following the same layering and
contract.

- **Layering per module:** handler → service → repository. Handlers are thin
  (bind, authorize, render the envelope); services own business logic and take
  `context.Context`; repositories own persistence.
- **Interface contracts at the boundary:** a handler depends on a small
  service-shaped interface it declares, not on a concrete type — so services are
  swappable and testable with fakes (as the auth reset/email tests show).
- **Shared building blocks** live in `internal/platform/*` and `internal/store/*`
  (envelope, errors, middleware, cache, resilience, generic repository/service).
- **BaseService hooks:** `BaseService[T]` exposes pre/post hooks (e.g.
  `PreCreate` for email-uniqueness + password hashing) so modules customize
  behavior without rewriting CRUD.
- **Module registration:** each module implements
  `RegisterRoutes(public, protected, admin)` and is wired in `cmd/server/main.go`.

## Consequences

- **+** Clear seams: a module is a self-contained handler+service+repo that
  composes shared primitives; forkers delete `items` and add their own.
- **+** One deployable — no distributed-systems tax (network hops, cross-service
  transactions) until a module genuinely needs to be extracted.
- **+** Interface-at-the-boundary keeps modules unit-testable without a database.
- **−** No enforced compile-time barrier stops one module importing another's
  internals; discipline (and review) maintains the boundary.
- **−** Everything scales together as one process; a hot module can't scale
  independently until it is extracted into its own service.

## Usage

- `internal/modules/items/` — reference module (delete when forking).
- `internal/modules/auth/` — JWT, API keys, RBAC, password reset.
- `internal/store/service.go` — `BaseService[T]` + hook interfaces.
- `cmd/server/main.go` — module construction and wiring.
