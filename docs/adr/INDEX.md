# Architecture Decision Records

One line per decision. Read the file for Context / Decision / Consequences / Usage.

| ADR | Decision | Status |
|---|---|---|
| [0001](0001-router-choice.md) | HTTP router: Gin | Accepted |
| [0002](0002-response-envelope.md) | Unified JSON response envelope | Accepted |
| [0003](0003-error-handling.md) | Typed errors with stable codes | Accepted |
| [0004](0004-resilience-scope.md) | Resilience scope: in-process for v0.1.0 (two-tier → v0.2) | Accepted |
| [0005](0005-db-driver-query-gen.md) | pgx + golang-migrate + sqlc + generic repository | Accepted |
| [0006](0006-modular-monolith.md) | Modular monolith architecture | Accepted |
