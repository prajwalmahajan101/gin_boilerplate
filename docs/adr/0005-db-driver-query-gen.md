# 0005 — Database: pgx + golang-migrate + sqlc + generic repository

**Status:** Accepted · 2026-10 · v0.1.0

## Context

The persistence layer needs a Postgres driver, a migration tool, and a query
strategy that is type-safe without becoming a heavyweight ORM. Options spanned
`database/sql` + `lib/pq`, `pgx`, ORMs (GORM, ent), hand-written SQL, and
codegen (sqlc). The boilerplate favors explicit SQL and compile-time safety over
ORM magic, while still removing CRUD boilerplate for simple domain modules.

## Decision

- **Driver:** `pgx` (v5) with `pgxpool` for connection pooling — native Postgres
  protocol, better performance and type support than `database/sql` + `lib/pq`.
- **Migrations:** `golang-migrate`, run with a Postgres **advisory lock** so
  concurrent instances don't race applying migrations. SQL migration files live
  in `migrations/` and are embedded.
- **Query generation:** `sqlc` compiles hand-written SQL into type-safe Go
  (`internal/store/db`), giving compile-time-checked queries without an ORM.
- **Generic CRUD:** a generic `Repository[T]` + `BaseService[T]` provide
  create/read/update/list/soft-delete for any model implementing the base
  contract, so a new domain module is handler + a few queries, not another
  hand-rolled CRUD layer.

## Consequences

- **+** Queries are explicit SQL, checked at build time by sqlc; no ORM runtime
  surprises or N+1 hidden behind lazy loading.
- **+** `Repository[T]`/`BaseService[T]` remove repetitive CRUD; modules compose
  them and add only domain-specific queries.
- **+** Advisory-locked migrations are safe to run from every booting instance.
- **−** sqlc requires a regeneration step (`make sqlc`) whenever queries change —
  generated code is committed.
- **−** The generic repository constrains models to the shared base-model shape
  (id, timestamps, soft-delete); genuinely different storage shapes still need
  bespoke repositories.

## Usage

- `internal/store/repository.go` (`Repository[T]`), `internal/store/service.go`
  (`BaseService[T]` with hooks), `internal/store/basemodel.go`.
- `internal/store/db/` — sqlc-generated queries and models.
- `migrations/` + `cmd/migrate` — golang-migrate with the advisory lock.
- Regenerate with `make sqlc`; apply migrations with `make migrate-up`.
