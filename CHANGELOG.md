# Changelog

All notable changes to this project are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While on `0.x`, the public API may change between minor versions.

## [Unreleased]

## [0.1.0] - 2026-10-09

First release — a production-shaped, forkable Gin REST boilerplate (the Go leg of
the Django / FastAPI / Gin trio).

### Added

- **Core foundation** — env-based config (with AWS Secrets Manager overlay),
  `slog` structured logging with request-id correlation, typed `apperr` errors
  with stable codes, a unified JSON response envelope
  (`{success, message, data, errors, request_id}`), generic `Repository[T]` and
  `BaseService[T]` with pre/post hooks and soft-delete, and pagination.
- **Middleware stack** — CORS, security headers (CSP/HSTS), body-size limit,
  panic recovery, request-id, request logging, and two-tier rate limiting.
- **HTTP server** — Gin engine with a three-gate router (public / protected /
  admin), graceful shutdown, `/healthz` + `/readyz` probes, and Swagger UI.
- **Database** — Postgres via pgx + pgxpool, `golang-migrate` with an advisory
  lock, and sqlc-generated typed queries.
- **Items module** — example CRUD domain module wired through
  handler → service → repository.
- **Auth module** — JWT access/refresh tokens, API keys, RBAC, user/role/
  permission management, a Valkey-backed token blacklist, logout, password
  change, and a forgot/reset-password flow over SES with per-user session
  revocation.
- **Cache stack** — tiered cache (L1 LRU → L2 Valkey → origin) with singleflight,
  negative caching, TTL jitter, and a fail-open circuit breaker.
- **Resilience primitives** — in-process circuit breaker, retry with exponential
  backoff + full jitter, an SSRF guard (`AssertPublicURL`), error classification,
  and a composed policy executor.
- **Crypto + AWS** — AES-256-GCM field encryption, and S3 / SES helpers over a
  shared AWS config loader.
- **Tooling & docs** — golangci-lint, pre-commit hooks, GitHub Actions CI,
  GoReleaser, k6 smoke + load tests, six ADRs, and a root `CLAUDE.md`.

### Notes

- Resilience is **in-process** for v0.1.0; distributed / two-tier coordination is
  planned for v0.2 (see [ADR 0004](docs/adr/0004-resilience-scope.md)).

[Unreleased]: https://github.com/prajwalmahajan101/gin_boilerplate/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/prajwalmahajan101/gin_boilerplate/releases/tag/v0.1.0
