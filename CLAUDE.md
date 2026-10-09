# CLAUDE.md

Guidance for working in this repository. Keep it current when conventions change.

## What this is

`gin_boilerplate` — a production-shaped, forkable Gin REST starter (third of the
Django / FastAPI / Gin trio). A **modular monolith**: one binary, domain modules
under `internal/modules/` composing shared platform primitives.

## Stack

Go 1.26 · Gin · pgx v5 + pgxpool · sqlc · golang-migrate · golang-jwt · go-redis
(Valkey) · slog · golangci-lint · k6 · Docker (distroless) · GoReleaser.

## Commands

```bash
make build        # compile to bin/server
make run          # build + run
make dev          # hot-reload (air)
make test         # go test -race ./...
make lint         # golangci-lint, --build-tags integration
make vet          # go vet -tags integration ./...
make fmt          # gofmt -w .
make check        # vet + lint + test (full verification gate)
make swagger      # regenerate OpenAPI spec (docs/swagger)
make sqlc         # regenerate query code from SQL
make migrate-up   # apply migrations (migrate-down n=N to roll back)
make migrate-create name=add_foo
make seed         # seed admin user (needs SEED_ADMIN_PASSWORD)
make load-smoke   # k6 smoke test (server must be up + migrated)
make load VUS=50 DURATION=2m   # k6 load test
make vuln         # govulncheck
make hooks        # install pre-commit hooks
```

## Layout

```
cmd/server        entry point + wiring      cmd/migrate, cmd/seed  CLIs
internal/
  platform/       cross-cutting infra: apperr, response, middleware, httpserver,
                  logger, cache, resilience, ratelimit, crypto, reqcontext, ...
  aws/            S3 + SES helpers over a shared config loader
  store/          generic Repository[T] + BaseService[T] + sqlc db/
  modules/        domain modules: items (example), auth (JWT/API key/RBAC/reset)
  valkey/         go-redis client      config/  env-based config (+ Secrets Manager)
migrations/       golang-migrate SQL    loadtest/  k6 scripts    docs/adr/  ADRs
```

## Conventions

- **Layering:** handler → service → repository. Handlers are thin (bind,
  authorize, render envelope). Services hold business logic and take
  `context.Context` (never `*gin.Context`). Repositories own persistence.
- **Interface at the boundary:** a handler declares the small service interface it
  needs; depend on that, not a concrete type (keeps modules fake-testable).
- **Response envelope:** every response goes through `response.Success` /
  `response.Error` — `{success, message, data, errors, request_id}`. See ADR 0002.
- **Errors:** return typed `apperr.*` with stable codes; handlers pass them to
  `response.Error`. Never return bare `errors.New` from a service boundary. ADR 0003.
- **New module:** mirror `internal/modules/items` — handler + service (compose
  `BaseService[T]`) + queries; implement `RegisterRoutes(public, protected, admin)`
  and wire in `cmd/server/main.go`.
- **Commits:** Conventional Commits (`feat|fix|refactor|docs|test|chore`), subject
  ≤72 chars, atomic. Never commit to `main` — feature branch + PR + green CI. No
  AI-attribution footer. Merge PRs with `--merge` (preserve history), not squash.
- **Config:** add env vars to `internal/config/config.go` (with `envDefault`) and
  mirror them in `.env.example`; redact secrets in `LogValue`.

## Gotchas

- **Integration build tag:** integration tests use `//go:build integration`. Lint
  and vet run with `-tags integration` (so `make check` compiles them) — a change
  that breaks an integration-tagged file won't show up under a plain `go build`.
- **Valkey is optional / fail-open:** empty `VALKEY_URL` disables it; the cache
  and blacklist degrade gracefully rather than erroring requests.
- **govulncheck pre-push hook + mise:** the pre-push `govulncheck` hook can fail to
  resolve its binary through the mise shim (`No version is set for shim`). Run
  `go run golang.org/x/vuln/cmd/govulncheck@latest ./...` manually; if clean, push
  with `--no-verify`. CI runs govulncheck independently.
- **Rate limits + load tests:** the public (login) and api gates are rate-limited.
  Run load tests against a server started with `RATE_LIMIT_RPM` /
  `LOGIN_RATE_LIMIT_RPM` raised; `loadtest/load.js` registers once in `setup()`.
- **Swagger is generated:** regenerate with `make swagger` after changing handler
  godoc; `docs/swagger` is committed.

## Architecture Decisions

Bodies in `docs/adr/`; index in `docs/adr/INDEX.md`.

- 0001 — HTTP router: Gin
- 0002 — Unified JSON response envelope
- 0003 — Typed errors with stable codes
- 0004 — Resilience scope: in-process for v0.1.0 (two-tier → v0.2)
- 0005 — pgx + golang-migrate + sqlc + generic repository
- 0006 — Modular monolith architecture
