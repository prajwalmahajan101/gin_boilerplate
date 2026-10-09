# gin_boilerplate -- Extraction Roadmap

> Bottom-up: quality gates first, core foundation second, modules plug in after. Every PR merges green.

**Legend:**
- `[BA]` = busy-api (main branch)
- `[BA-L]` = busy-api (legacy branch)
- `[DS]` = document-service
- `[DJ]` = django_boilerplate (pattern reference -- Python)
- `[FA]` = fastapi_boilerplate (pattern reference -- Python)
- `[NEW]` = write fresh for gin_boilerplate

**Architecture: Modular Monolith**

```
cmd/server/main.go          -- wires modules, starts server
internal/
  platform/                  -- cross-cutting infrastructure (no domain knowledge)
    apperr/                  -- typed errors + error code registry
    response/                -- JSON envelope (Success/Error/Paginated)
    middleware/               -- HTTP middleware stack
    httpserver/               -- server lifecycle, router, probes
    logger/                  -- slog setup, request-id injection
    reqcontext/              -- request-scoped context carriers
    pagination/              -- page/size extraction + meta
    cache/                   -- tiered cache (L1 LRU -> L2 Valkey -> DB)
    resilience/              -- breaker, retry, SSRF, executor
    crypto/                  -- AES-256-GCM field encryption
  store/                     -- generic persistence layer
    basemodel.go             -- BaseModel (id, is_active, timestamps)
    repository.go            -- Repository[T]: generic CRUD + soft-delete + pagination
    service.go               -- BaseService[T]: generic CRUD + pre/post hooks + cascade
    pool.go                  -- pgxpool init + per-call timeout
    migrate.go               -- golang-migrate runner
  modules/                   -- domain modules (each self-contained)
    items/                   -- example CRUD (handler/service/repo) -- delete when forking
    auth/                    -- JWT, API key, RBAC, users, roles, permissions
  valkey/                    -- go-redis client factory
  aws/                       -- S3, SES helpers
  config/                    -- env-based config + Secrets Manager
```

**Module rules:**
- Each module owns its handler, service, repository, models
- Modules depend on each other via **interfaces defined by the consumer** (not concrete types)
- Platform packages never import modules; modules import platform
- `store/` provides generic base types; modules embed/compose them

---

## M0: Quality Gates + Project Skeleton ✅

**Goal:** Every subsequent PR has green CI before merge. Linting, formatting, pre-commit hooks, CI pipeline, goreleaser, and project skeleton -- all in the first commit to `main`.

### Extract

| Target file | Source | Action |
|---|---|---|
| `go.mod` | `[NEW]` | `go mod init github.com/prajwalch/gin_boilerplate`. Seed minimal `cmd/server/main.go` + placeholder package so linter has something to check. |
| `.golangci.yml` | Merge `[BA]` + `[DS]` | DS's richer linter set: errcheck, gocritic, gosec, govet, ineffassign, misspell, revive, staticcheck, unused. BA's gosec exclusions (G304, G107). DS's gocritic + revive rules. golangci-lint v2 format. |
| `.githooks/pre-commit` | `[DS] .githooks/pre-commit` | Copy verbatim. `gofmt -l` + `go vet ./...` on staged .go files. |
| `.github/workflows/ci.yml` | Merge `[BA]` + `[DS]` | Four jobs: (1) build-vet-test: gofmt check + go vet + `go test -race -timeout 5m`; (2) lint: golangci-lint-action v8; (3) vuln: govulncheck; (4) docker: image build. PR-to-main trigger. Concurrency cancel. Services block for Postgres 16 + Valkey 8 (from M4 onward). |
| `.github/workflows/release.yml` | `[NEW]` | GoReleaser action on tag push `v*`. |
| `.goreleaser.yml` | `[NEW]` | Builds `cmd/server` for linux/darwin amd64+arm64. Docker from distroless. Changelog from conventional commits. |
| `Makefile` | `[BA] Makefile` | Core: `build`, `run`, `dev`, `test`, `lint`, `fmt`, `vet`, `hooks`, `vuln`. Stubs: `migrate-*`, `sqlc`, `swagger`, `compose-*`, `load-*`. |
| `.air.toml` | `[BA] .air.toml` | Copy. Adjust build path. |
| `.env.example` | `[NEW]` | Seed: `PORT`, `ENV`, `LOG_LEVEL`, `LOG_JSON`. Grow per milestone. |
| `.gitignore` | `[BA]` | + `tmp/`, `bin/`, `.env`, `dist/`, `docs/swagger/` |
| `Dockerfile` | `[DS] Dockerfile` | Multi-stage: golang:1.22-alpine -> distroless. Non-root. CGO_ENABLED=0. |
| `docker-compose.yml` | Merge `[BA]` + `[DS]` | Postgres 16 + Valkey 8 + app. Healthchecks. |
| `README.md` | `[NEW]` | Minimal WIP. Grows per milestone. |

### Verify

- [x] `make lint` / `make test` / `make vet` / `make fmt` all pass
- [x] `make hooks` installs pre-commit; unformatted .go -> rejected
- [x] `make vuln` passes
- [x] `go build ./cmd/server` compiles
- [x] `docker build .` works
- [x] CI green on PR (all 4 jobs)
- [x] `goreleaser check` valid

**No code PR merges without green CI from this point forward.**

---

## M1: Core Foundation ✅

**Goal:** All shared patterns that every module depends on. This is the heart of the boilerplate -- the "core" package equivalent from Django/FastAPI. Config, logging, errors, envelope, base model, generic repository, base service with hooks, pagination. After this milestone, a new domain module is just handler + service + repo that composes these building blocks.

### M1a: Config + Logging + Request Context

| Target file | Source | Action |
|---|---|---|
| `internal/config/config.go` | `[BA] internal/config/config.go` | Config struct + `Load()` via `caarlos0/env` + `joho/godotenv`. Start with server + logging + DB + Valkey fields (all fields up front so later milestones just wire them). |
| `internal/config/secrets.go` | `[DS] internal/config/secrets.go` | `LoadSecretsBundle()` -- AWS Secrets Manager overlay. Optional. |
| `internal/platform/reqcontext/reqcontext.go` | `[BA] internal/reqcontext/reqcontext.go` | Copy. Cycle-free. `WithRequestID` / `RequestIDFromContext`. |
| `internal/platform/reqcontext/timing.go` | `[BA] internal/reqcontext/timing.go` | Copy. Per-layer timing: `TrackHandler`, `TrackService`, `TrackRepo`. |
| `internal/platform/logger/logger.go` | Merge `[BA] logging.go` + `[DS] logger.go` | `Setup()`, `ctxHandler` (request_id injection), convenience wrappers, `ReplaceAttr` sanitization. slog JSON + lumberjack rotation. |

### M1b: Typed Errors + Response Envelope

| Target file | Source | Action |
|---|---|---|
| `internal/platform/apperr/apperr.go` | Merge `[BA] errs.go` + `[DS] apperr.go` | `AppError` struct (Code, Message, HTTPStatus, Details, RequestID, trips). Full code registry: `VALIDATION_ERROR` (400), `INVALID_INPUT` (400), `UNAUTHORIZED` (401), `FORBIDDEN` (403), `NOT_FOUND` (404), `CONFLICT` (409), `PAYLOAD_TOO_LARGE` (413), `RATE_LIMITED` (429), `INTERNAL_ERROR` (500), `EXTERNAL_ERROR` (502), `TRANSIENT_ERROR` (502), `TIMEOUT` (502), `SERVICE_UNAVAILABLE` (503). Helpers: `New()`, `Newf()`, `Wrap()`, `Is()`, `AsAppError()`, `HTTPStatus()`, `TripsBreaker()`. |
| `internal/platform/response/response.go` | Merge `[BA] response.go` + `[DS] response.go` | `Envelope` (success, message, data, errors, request_id). `ErrDetail` (code, message, field, details). `Success()`, `Error()`, `Paginated()`, `ErrorHandler()` middleware. |
| `internal/platform/pagination/pagination.go` | `[DS] pagination.go` | `Meta` (page, page_size, total, total_pages, has_next, has_prev). `Params(c)` extractor. `MetaFromQuery()`. |

### M1c: Base Model + Generic Repository

| Target file | Source | Action |
|---|---|---|
| `internal/store/basemodel.go` | `[DS] basemodel.go` | `BaseModel` struct: ID (int64), IsActive (bool), CreatedAt, UpdatedAt (time.Time). `Model` interface: `TableName() string`. |
| `internal/store/pool.go` | `[BA] db.go` | `NewPool()` -- pgxpool sized by config, ping, per-call `WithQueryTimeout()`. `Atomic(ctx, pool, fn)` transaction wrapper. |
| `internal/store/repository.go` | Merge `[DS] repository.go` + `[BA] repo.go` | `Querier` interface (Query/QueryRow/Exec -- satisfied by both `*pgxpool.Pool` and `pgx.Tx`). `Repository[T Model]` with: `Create(ctx, *T)`, `FindByID(ctx, id)`, `FindActiveByID(ctx, id)`, `List(ctx, page, size)` (with clamping + overflow guard), `Update(ctx, *T)`, `SoftDelete(ctx, id)`, `HardDelete(ctx, id)`, `Exists(ctx, id)`, `Count(ctx)`, `WithTx(tx)`. Row scanning via `RowToStructByNameLax` + `db` struct tags from DS. Pagination helpers from BA (page/size clamping, MaxPageSize=1000, int32 guard). |

### M1d: Base Service with Hooks

| Target file | Source | Action |
|---|---|---|
| `internal/store/service.go` | `[NEW]` based on `[DJ] core/base/service.py` + `[FA] core/base/service.py` | **This is new Go code porting the Python BaseService pattern.** `BaseService[T Model]` generic struct wrapping a `Repository[T]`. **CRUD methods**: `Create(ctx, data)` -- calls `PreCreate` hook, repo.Create, `PostCreate` hook. `Update(ctx, id, data)` -- calls `PreUpdate`, repo.Update, `PostUpdate`. `Delete(ctx, id, soft bool)` -- calls `PreDelete`, repo.SoftDelete or HardDelete, `PostDelete`, optional cascade. `GetByID(ctx, id)` / `GetActiveByID(ctx, id)` / `GetByIDOrFail(ctx, id)` -- wraps repo with `NOT_FOUND` error. `List(ctx, page, size)` -- delegates to repo + returns pagination Meta. **Hooks interface**: `ServiceHooks[T]` with `PreCreate(ctx, *T) error`, `PostCreate(ctx, *T)`, `PreUpdate(ctx, *T) error`, `PostUpdate(ctx, *T)`, `PreDelete(ctx, *T) error`, `PostDelete(ctx, *T)`. Default no-op implementation embedded. Concrete services override only the hooks they need. **Soft-delete cascade**: `CascadeSoftDelete(ctx, parentID, children []CascadeTarget)` -- breadth-first walk with depth cap (MAX_DEPTH=10). `CascadeTarget` interface: `SoftDeleteByParent(ctx, parentID) error`. Pattern from `[DJ] _cascade_soft_delete_bfs()`. **Transaction boundary**: Service methods that write use `store.Atomic()` -- transactions owned at service layer (matches DJ pattern where routes wrap in `@transaction.atomic`). |

### M1e: Module Wiring Pattern

| Target file | Source | Action |
|---|---|---|
| `internal/modules/modules.go` | `[NEW]` | Module registration pattern. `Module` interface: `RegisterRoutes(public, protected, admin *gin.RouterGroup)`. Each domain module implements this. `cmd/server/main.go` collects modules and calls `RegisterRoutes` on each. Interface contracts: modules that need cross-module access declare a consumer-side interface (e.g., `items` module declares `type Authenticator interface { ... }` if it needs auth, not import auth.Service). |

### Verify

- [x] Config loads all fields from `.env` with sensible defaults
- [x] Logger outputs JSON with request_id on `slog.InfoContext`
- [x] `apperr.New(apperr.CodeNotFound, "x")` -> status 404, code `NOT_FOUND`
- [x] `response.Success()` / `response.Error()` / `response.Paginated()` produce correct envelope JSON
- [x] `BaseModel` fields present: id, is_active, created_at, updated_at
- [x] `Repository[T].Create()` + `FindByID()` + `List()` + `SoftDelete()` compile (unit test with mock Querier)
- [x] `BaseService[T]` CRUD calls hooks in correct order (unit test with hook counter)
- [x] `CascadeSoftDelete` walks children with depth cap (unit test)
- [x] `Module` interface compiles, module registration pattern works
- [x] All pagination edge cases: page < 1 clamped, size > MaxPageSize clamped, int32 overflow guarded
- [x] `make test` + `make lint` green
- [x] CI green on PR

---

## M2: Middleware Stack ✅

**Goal:** Full middleware chain. No auth or rate-limit yet (those come with their modules).

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/platform/middleware/middleware.go` | `[BA] middleware.go` | `Setup()` -- ordered chain: Recovery -> BodyLimit -> CORS -> SecurityHeaders -> RequestID -> RequestLogging -> ErrorHandler. Auth + rate-limit added when auth module lands. |
| `internal/platform/middleware/recovery.go` | `[BA]` | Broken-pipe detection, stack logging, 500 envelope. |
| `internal/platform/middleware/body_limit.go` | `[BA]` | Content-Length pre-check + MaxBytesReader + 413 envelope. |
| `internal/platform/middleware/cors.go` | `[BA]` | Map-based allowlist, 204 preflight, 403 disallowed. |
| `internal/platform/middleware/security_headers.go` | `[BA]` | CSP, HSTS (prod only), X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy. |
| `internal/platform/middleware/request_id.go` | `[BA]` | Regex validation, UUID mint, context inject, header echo. |
| `internal/platform/middleware/request_logging.go` | `[BA]` | `timingWriter` for Server-Timing, structured access log. |

### Verify

- [x] Panic -> 500 envelope with request_id
- [x] Oversized body -> 413 envelope
- [x] Every response has `X-Request-ID` header
- [x] Security headers on every response
- [x] CORS preflight 204 / 403
- [x] CI green on PR

---

## M3: HTTP Server + Health Probes + Swagger + Docker ✅

**Goal:** Gin engine with three-gate router (public/protected/admin), graceful shutdown, health probes, Swagger UI, Docker Compose stack. First runnable server.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/platform/httpserver/server.go` | `[DS]` | `Server` + `RunWithGracefulShutdown(ctx)`. |
| `internal/platform/httpserver/router.go` | `[DS]` | Three-gate: `RouterConfig` + `NewRouter()`. Public/protected/admin groups. Swagger on `/swagger/*`. Modules register via `Module.RegisterRoutes()`. |
| `internal/platform/httpserver/probes.go` | `[DS]` | `Liveness()` -> 200. `Readiness(postgres, valkey Check)` -> soft-dep Valkey. |
| `internal/platform/httpserver/clientip.go` | `[DS]` | X-Forwarded-For + CIDR trust. |
| `docs/swagger/` | `[NEW]` | `swag init`. Wire gin-swagger. Top-level annotations in main.go. |
| `cmd/server/main.go` | Merge `[BA]` + `[DS]` | `run()` pattern + graceful shutdown + module registration loop. |

### Verify

- [x] `docker compose up -d` brings up Postgres + Valkey + app
- [x] `curl /healthz` -> 200
- [x] `curl /readyz` -> postgres + valkey checks
- [x] `curl /swagger/index.html` -> Swagger UI
- [x] SIGTERM -> graceful drain
- [x] CI green on PR

---

## M4: Database Layer + Migrations ✅

**Goal:** Wire generic Repository[T] to real Postgres via pgxpool. golang-migrate with advisory lock. sqlc codegen. First real DB round-trip.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/store/migrate.go` | `[DS]` | golang-migrate with `embed` + advisory lock for concurrent boots. |
| `internal/store/db/` | `[NEW]` via sqlc | Generated from `internal/store/queries/*.sql`. |
| `internal/store/queries/items.sql` | `[BA]` | Item CRUD queries adapted to PRD schema (name, code, notes). |
| `sqlc.yaml` | `[BA]` | Adjusted paths. |
| `migrations/000001_base.up.sql` | `[NEW]` | Items table per PRD 11.2. |
| `migrations/000001_base.down.sql` | `[NEW]` | `DROP TABLE items;` |

### Verify

- [x] `make migrate-up` creates items table
- [x] `make sqlc` generates Go code
- [x] `Repository[Item].Create()` + `FindByID()` round-trips against real Postgres (integration test)
- [x] `BaseService[Item]` hooks fire during real CRUD (integration test)
- [x] `SoftDelete` sets `is_active = false` (integration test)
- [x] Per-call query timeout fires on slow queries
- [x] `make migrate-down` rolls back cleanly
- [x] CI green on PR (integration tests against service containers)

---

## M5: Items Module (Example Domain) ✅

**Goal:** First domain module plugged into the modular monolith. Full handler -> service -> repository using the core BaseService[T]. End-to-end CRUD via REST.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/modules/items/handler.go` | `[BA] items/handler.go` | `Handler` + `RegisterRoutes()` (implements `Module` interface). DTOs: `itemDTO`, `createReq`, `updateReq`. Routes: `/api/v1/items`. Swag annotations per handler. |
| `internal/modules/items/service.go` | `[BA] items/service.go` | `ItemService` embeds `BaseService[Item]`. Overrides `PreCreate` hook for code-uniqueness validation. `Update()` method. No cache logic yet (M8). |
| `internal/modules/items/repository.go` | `[NEW]` | Embeds `Repository[Item]`. Adds `GetByCode(ctx, code)` custom method via sqlc. |
| `internal/modules/items/model.go` | `[NEW]` | `Item` struct embedding `BaseModel` + Name, Code, Notes. `TableName() = "items"`. |

### Verify

- [x] `POST /api/v1/items` -> 201 + envelope
- [x] `GET /api/v1/items` -> 200 + paginated
- [x] `GET /api/v1/items/:id` -> 200 + item
- [x] `PATCH /api/v1/items/:id` -> 200 + updated
- [x] `DELETE /api/v1/items/:id` -> 200 + soft-deleted
- [x] `PreCreate` hook rejects duplicate code -> 409 `CONFLICT`
- [x] Invalid body -> 400 `VALIDATION_ERROR`
- [x] Missing item -> 404 `NOT_FOUND`
- [x] Request-ID in every response + log
- [x] Swagger UI shows all item endpoints
- [x] CI green on PR

---

## M6: Valkey Client + Auth Module ✅

**Goal:** Valkey client (used by auth for JWT blacklist + rate-limit). Auth module: JWT, API key, RBAC, users, roles, permissions. Two-tier rate-limit middleware. Second domain module proving the modular monolith pattern.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/valkey/client.go` | Merge `[BA]` + `[DS]` | go-redis client. `Incr`, `Get`, `SetEX`, `SetNX`, `Del`, `Ping`. Nil when disabled. |
| `internal/platform/middleware/auth.go` | `[DS] middleware.go` (Auth) | JWT + API key resolver. Sets user context (user_id, user_name, is_superuser). |
| `internal/platform/middleware/rate_limit.go` | `[DS] middleware_ratelimit.go` | Two-tier: `RateLimiter` interface + `WithFallback(primary, fallback)`. IP-keyed public, entity-keyed protected. |
| `internal/platform/ratelimit/ratelimit.go` | `[DS]` | `MemoryCounter` -- in-process fixed-window fallback. |
| `migrations/000002_auth.up.sql` | `[NEW]` | Users, roles, permissions, user_roles, role_permissions, api_keys per PRD 11.1. |
| `internal/store/queries/auth.sql` | `[NEW]` | sqlc queries for auth tables. |
| `internal/modules/auth/model.go` | Merge `[DS] entity/model.go` + `[DJ]`/`[FA]` RBAC | User, Role, Permission, APIKey. All embed BaseModel. |
| `internal/modules/auth/password.go` | `[DS] entity/password.go` | bcrypt hash/verify. |
| `internal/modules/auth/jwt.go` | `[NEW]` based on `[DS] authtoken.go` + `[FA]` | `golang-jwt/jwt/v5`. Access + refresh with `jti`. Blacklist via Valkey. Fail-open access, fail-closed refresh. |
| `internal/modules/auth/apikey.go` | `[DS] entity/keys.go` | `Generate()`, `Hash()` (HMAC-SHA256 + pepper). Prefix-based DB lookup. |
| `internal/modules/auth/rbac.go` | `[NEW]` based on `[DJ]` | `HasPermission(ctx, userID, resource, action)`. Superuser short-circuit. Per-request cache on context. |
| `internal/modules/auth/repository.go` | Merge `[DS] entity/repo.go` + `[NEW]` | Embeds `Repository[User]`. `GetByEmail()`, `ResolveAPIKey()`, role/permission queries. |
| `internal/modules/auth/service.go` | Merge `[DS] entity/service.go` + `[NEW]` | `AuthService` embeds `BaseService[User]`. Login, refresh, logout, API key identity, user/role/permission management. Overrides `PreCreate` hook for email uniqueness + password hashing. |
| `internal/modules/auth/handler.go` | Merge `[DS] entity/handlers.go` + `admin_handlers.go` | Public: login, refresh. Protected: logout. Admin: user CRUD, API key issue, role/permission management. Swag annotations. Implements `Module` interface. |

**Interface contract example:** If items module needs to check auth, it declares:
```go
// internal/modules/items/service.go
type Authenticator interface {
    HasPermission(ctx context.Context, userID int64, resource, action string) bool
}
```
Auth module satisfies this interface. Wired in `cmd/server/main.go`.

### Verify

- [x] `POST /auth/login` -> 200 + tokens
- [x] Bad creds -> 401 `UNAUTHORIZED`
- [x] Items without token -> 401
- [x] Items with token -> 200
- [x] Refresh -> new pair
- [x] Logout -> blacklisted
- [x] API key via `X-API-Key` -> authenticated
- [x] User without permission -> 403 `FORBIDDEN`
- [x] Admin endpoints require superuser
- [x] Rate limit exceeded -> 429 `RATE_LIMITED`
- [x] Rate limit falls back to memory when Valkey down
- [x] Swagger shows all auth endpoints
- [x] `BaseService[User].PreCreate` fires (password hash, email check)
- [x] CI green on PR

---

## M7: Cache Stack ✅

**Goal:** Tiered cache, singleflight, negative cache, TTL jitter. Wire into items service.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/platform/cache/cache.go` | `[BA]` | `Cache` interface: Get, Set, Del. |
| `internal/platform/cache/memory.go` | `[BA]` | In-process LRU + TTL. |
| `internal/platform/cache/lru.go` | `[BA]` | hashicorp/golang-lru wrapper. |
| `internal/platform/cache/valkey.go` | `[BA]` | Valkey backend. |
| `internal/platform/cache/tiered.go` | `[BA]` | L1 -> L2 -> origin. |
| `internal/platform/cache/breaker_cache.go` | `[BA]` | Breaker-guarded Valkey. |
| `internal/platform/cache/failopen.go` | `[BA]` | Errors -> miss. |
| `internal/platform/cache/jitter.go` | `[BA]` | TTL +/- % spread. |
| `internal/modules/items/service.go` | `[BA]` | Add cache-aside: singleflight + cache get/set + negative cache. |

### Verify

- [x] Second GET hits cache (no DB query)
- [x] Valkey down -> fail-open, items still served
- [x] Concurrent GETs -> singleflight (1 DB read)
- [x] Non-existent ID -> negative cache
- [x] TTLs have jitter
- [x] CI green on PR

---

## M8: Resilience Primitives ✅

**Goal:** Circuit breaker, retry, SSRF guard, composed executor.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/platform/resilience/breaker.go` | `[BA] resilience/breaker/*.go` | In-process: CLOSED -> OPEN -> HALF_OPEN. |
| `internal/platform/resilience/retry.go` | `[BA-L] resilience/retry/` | Exponential backoff + full jitter. |
| `internal/platform/resilience/ssrf.go` | `[DS]` | `AssertPublicURL()`. |
| `internal/platform/resilience/classify.go` | `[DS]` | `Retryable(err)`. |
| `internal/platform/resilience/executor.go` | `[DS]` | Composed policy: rate-limit -> breaker -> timeout -> retry. |

### Verify

- [x] Breaker opens/half-opens correctly
- [x] Retry backs off with jitter
- [x] SSRF rejects private/loopback/link-local
- [x] Executor composes correctly
- [x] CI green on PR

---

## M9: Crypto + AWS ✅

**Goal:** AES-256-GCM field encryption, S3/SES helpers.

### Extract

| Target file | Source | Action |
|---|---|---|
| `internal/platform/crypto/crypto.go` | `[DS]` | AES-256-GCM encrypt/decrypt. |
| `internal/aws/s3.go` | `[NEW]` based on `[DS] s3store.go` | `Put()`, `PresignGet()`. |
| `internal/aws/ses.go` | `[NEW]` based on `[DJ]` | `SendEmail(to, subject, body)`. |

### Verify

- [x] Encrypt/decrypt round-trips
- [x] Wrong key -> fails
- [x] CI green on PR

---

## M10: Load Tests + ADRs + v0.1.0

**Goal:** k6 tests, ADRs, final polish, tag.

### Extract

| Target file | Source | Action |
|---|---|---|
| `loadtest/smoke.js` | `[BA]` | Adapted to `/api/v1/items` + auth. |
| `loadtest/load.js` | `[BA]` | Parameterized. |
| `docs/adr/0001-router-choice.md` | `[NEW]` | gin. |
| `docs/adr/0002-response-envelope.md` | `[NEW]` | Single JSON shape. |
| `docs/adr/0003-error-handling.md` | `[NEW]` | Typed AppError + stable codes. |
| `docs/adr/0004-resilience-scope.md` | `[NEW]` | In-process v0.1.0, two-tier v0.2. |
| `docs/adr/0005-db-driver-query-gen.md` | `[NEW]` | golang-migrate + sqlc + generic repo. |
| `docs/adr/0006-modular-monolith.md` | `[NEW]` | Module boundaries, interface contracts, BaseService hooks. |
| `CLAUDE.md` | `[NEW]` | Stack, commands, conventions, gotchas. |
| `README.md` | Update | Full first-time flow: clone -> .env -> docker compose up -> CRUD. |

### Verify

- [x] `make load-smoke` passes
- [x] All 6 ADRs written
- [x] `go test -race ./...` clean
- [x] `golangci-lint run` clean
- [x] `goreleaser check` valid
- [x] README flow works end-to-end
- [x] Tag `v0.1.0`
- [x] Release artifacts published

---

## Dependency Graph

```
M0  (quality gates)         ✅  CI/lint/hooks/goreleaser -- gates everything
 |
M1  (core foundation)       ✅  config, logging, errors, envelope, base model,
 |                               generic repo, base service + hooks, pagination,
 |                               module wiring pattern
M2  (middleware stack)       ✅
 |
M3  (HTTP server + swagger) ✅
 |
M4  (DB + migrations)       ✅  base repo wired to real Postgres
 |
M5  (items module)           ✅  first domain module, proves the pattern
 |
M6  (valkey + auth module)   ✅  JWT, API key, RBAC, rate-limit w/ fallback
 |
M7  (cache stack)            ✅  performance layer on items
 |
M8  (resilience)             ✅  outbound safety
 |
M9  (crypto + AWS)           ✅  optional infrastructure
 |
M10 (load tests + tag)       ⬚  v0.1.0
```

**Rule: every PR -> main requires green CI. No exceptions.**

**Critical path**: M0 -> M1 -> M2 -> M3 -> M4 -> M5 (first working API).

**After M5**: M6 (auth) is next natural step. M7 (cache) needs M6's Valkey client. M8 (resilience) and M9 (crypto/AWS) are independent after M5.

---

## Source File Cross-Reference

| gin_boilerplate path | Primary source | Secondary source |
|---|---|---|
| `.golangci.yml` | `[DS]` | `[BA]` |
| `.github/workflows/ci.yml` | `[DS]` | `[BA]` |
| `.githooks/pre-commit` | `[DS]` | -- |
| `internal/config/config.go` | `[BA]` | `[DS]` |
| `internal/config/secrets.go` | `[DS]` | -- |
| `internal/platform/logger/logger.go` | `[BA]` | `[DS]` |
| `internal/platform/reqcontext/*.go` | `[BA]` | -- |
| `internal/platform/apperr/apperr.go` | `[BA]` | `[DS]` |
| `internal/platform/response/response.go` | `[BA]` | `[DS]` |
| `internal/platform/pagination/pagination.go` | `[DS]` | -- |
| `internal/platform/middleware/recovery.go` | `[BA]` | `[DS]` |
| `internal/platform/middleware/body_limit.go` | `[BA]` | `[DS]` |
| `internal/platform/middleware/cors.go` | `[BA]` | -- |
| `internal/platform/middleware/security_headers.go` | `[BA]` | `[DS]` |
| `internal/platform/middleware/request_id.go` | `[BA]` | `[DS]` |
| `internal/platform/middleware/request_logging.go` | `[BA]` | `[DS]` |
| `internal/platform/middleware/auth.go` | `[DS]` | -- |
| `internal/platform/middleware/rate_limit.go` | `[DS]` | -- |
| `internal/platform/httpserver/*.go` | `[DS]` | -- |
| `internal/platform/cache/*.go` | `[BA]` | -- |
| `internal/platform/resilience/breaker.go` | `[BA]` | -- |
| `internal/platform/resilience/retry.go` | `[BA-L]` | -- |
| `internal/platform/resilience/ssrf.go` | `[DS]` | -- |
| `internal/platform/resilience/classify.go` | `[DS]` | -- |
| `internal/platform/resilience/executor.go` | `[DS]` | -- |
| `internal/platform/crypto/crypto.go` | `[DS]` | -- |
| `internal/store/pool.go` | `[BA]` | -- |
| `internal/store/migrate.go` | `[DS]` | -- |
| `internal/store/basemodel.go` | `[DS]` | -- |
| `internal/store/repository.go` | `[DS]` | `[BA]` |
| `internal/store/service.go` | `[NEW]` | `[DJ]` + `[FA]` BaseService pattern |
| `internal/valkey/client.go` | `[BA]` | `[DS]` |
| `internal/modules/items/*.go` | `[BA] items/` | -- |
| `internal/modules/auth/model.go` | `[DS] entity/model.go` | `[DJ]`/`[FA]` RBAC |
| `internal/modules/auth/password.go` | `[DS] entity/password.go` | -- |
| `internal/modules/auth/jwt.go` | `[NEW]` | `[DS] authtoken.go` + `[FA]` |
| `internal/modules/auth/apikey.go` | `[DS] entity/keys.go` | -- |
| `internal/modules/auth/rbac.go` | `[NEW]` | `[DJ]` permission pattern |
| `internal/modules/auth/repository.go` | `[DS] entity/repo.go` | -- |
| `internal/modules/auth/service.go` | `[DS] entity/service.go` | -- |
| `internal/modules/auth/handler.go` | `[DS] entity/handlers.go` | `[DS] admin_handlers.go` |
| `cmd/server/main.go` | `[BA]` | `[DS]` |
