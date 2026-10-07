# gin_boilerplate — Product Requirements Document

> Production-shaped Gin REST starter. Third leg of the boilerplate trio (Django · FastAPI · Gin).

---

## 1. Problem

The portfolio carries production-shaped REST in Python only (`django_boilerplate` + `fastapi_boilerplate`). Go output is systems-level (toymq, toykv, ToyRaft) — no production REST surface. A reviewer correctly concludes: "Python is where Prajwal ships APIs."

A Go production REST boilerplate, opinionated the same way as the Python pair, says "I can ship production REST in three runtimes."

---

## 2. Goal

Forkable Go service skeleton with **the same opinions** as the Python boilerplates:
- Typed errors with stable error codes
- Unified response envelope
- Request-ID correlation end-to-end
- Structured logging (slog)
- Resilience primitives (retry, circuit breaker, SSRF guard)
- Handler → Service → Repository layering
- JWT + API key auth
- Soft-delete, audit fields, pagination
- OpenAPI, health probes, Docker, CI

**Not a framework.** A starter repo you fork and gut the example resource.

---

## 3. Reference Architecture

Patterns are drawn from four existing codebases. Every design choice below traces to a working implementation.

| Pattern | Django boilerplate | FastAPI boilerplate | busy-api | document-service |
|---|---|---|---|---|
| Response envelope | `core/base/response.py` | `core/responses/envelope.py` | `internal/response/response.go` | `platform/response/response.go` |
| Typed errors | `core/base/exception.py` | `core/exceptions/` | `internal/errs/errs.go` | `platform/apperr/apperr.go` |
| Middleware stack | Django middleware classes | Starlette middleware | `internal/middleware/` | `platform/httpserver/middleware*.go` |
| Layering | views → services → repos | routers → services → repos | handler → service → store | handler → service → repo |
| Auth | JWT + API key + OAuth2 | JWT + API key + OAuth2 | — (not yet) | HMAC bearer + API key + super-admin |
| Resilience | resilience-kit (Python) | resilience-kit (Python) | in-process breaker + cache | failsafe-go + two-tier state |
| Config | Django settings + env | Pydantic settings + env | caarlos0/env | caarlos0/env + Secrets Manager |
| DB | Django ORM + Postgres | SQLAlchemy + Alembic | pgx + sqlc + goose | pgx + golang-migrate |
| Soft-delete | BaseModel.is_active + cascade | BaseModel.is_active | items.is_active | BaseModel.is_active |
| Health probes | /health/ + /ready/ | /healthz + /readyz | /ping | /health + /ready |
| OpenAPI | drf-spectacular | FastAPI built-in | — | — |
| Docker | multi-stage + nginx + celery | multi-stage + uvicorn | docker-compose (Valkey) | multi-stage alpine + compose |

---

## 4. Stack

| Component | Choice | Why |
|---|---|---|
| Language | Go ≥ 1.22 | Generics stable; slog in stdlib |
| Router | `gin-gonic/gin` | Largest ecosystem; naming convention matches trio |
| DB driver | `jackc/pgx/v5` | Native Postgres, typed, pool-aware |
| Query gen | `sqlc` | SQL → Go; no runtime reflection |
| Migrations | `pressly/goose` | Proven in busy-api; SQL-based, embeddable |
| Cache | `redis/go-redis/v9` (Valkey-compatible) | Proven in busy-api + document-service |
| Logging | `log/slog` (stdlib) | No dependency; JSON structured output |
| Log rotation | `natefinch/lumberjack` | Proven in busy-api |
| Config | `caarlos0/env` + `joho/godotenv` | Proven in busy-api + document-service |
| Auth (JWT) | `golang-jwt/jwt/v5` | Standard Go JWT library |
| Auth (API key) | HMAC-SHA256 (stdlib crypto) | Pattern from document-service |
| Password hash | `golang.org/x/crypto/bcrypt` | Pattern from document-service |
| AWS | `aws-sdk-go-v2` | Secrets Manager + S3 + SES |
| OpenAPI | `swaggo/swag` | Annotation-based; generates from Gin handlers |
| Linter | `golangci-lint` + `gofumpt` | Standard Go quality gate |
| Hot reload | `air` | Proven in busy-api |
| Load test | `k6` | Proven in busy-api |
| Docker | Multi-stage (distroless, non-root) | Production standard |
| UUID | `google/uuid` | UUIDv4 for request IDs |
| Singleflight | `golang.org/x/sync/singleflight` | Cache stampede defense |
| LRU (L1) | `hashicorp/golang-lru/v2` | Proven in busy-api |

---

## 5. Project Structure

```
gin_boilerplate/
├── cmd/server/
│   └── main.go                         # Entry: wire deps, build router, serve, graceful shutdown
│
├── internal/
│   ├── config/
│   │   ├── config.go                   # Env-based config struct (caarlos0/env)
│   │   └── secrets.go                  # AWS Secrets Manager overlay (optional)
│   │
│   ├── platform/                       # Cross-cutting, domain-free infrastructure
│   │   ├── response/
│   │   │   └── response.go            # Envelope: Success / Error / Paginated
│   │   ├── apperr/
│   │   │   └── apperr.go              # Typed errors: AppError + stable codes + HTTP status registry
│   │   ├── middleware/
│   │   │   ├── middleware.go           # Ordered chain installer
│   │   │   ├── recovery.go            # Panic recovery → 500 envelope
│   │   │   ├── request_id.go          # Mint/adopt X-Request-ID
│   │   │   ├── request_logging.go     # Structured access log + Server-Timing
│   │   │   ├── cors.go                # Configurable CORS allowlist
│   │   │   ├── security_headers.go    # CSP, HSTS, X-Frame-Options
│   │   │   ├── body_limit.go          # MaxBytesReader → 413
│   │   │   ├── rate_limit.go          # IP-keyed + entity-keyed inbound limit
│   │   │   └── auth.go                # JWT + API key resolver → context
│   │   ├── httpserver/
│   │   │   ├── server.go              # HTTP server + graceful shutdown
│   │   │   ├── router.go              # Route groups: public / protected / admin
│   │   │   └── probes.go             # /healthz (liveness) + /readyz (readiness)
│   │   ├── logger/
│   │   │   └── logger.go             # slog setup, JSON, rotation, request-id context handler
│   │   ├── reqcontext/
│   │   │   ├── reqcontext.go          # Request-ID context carrier (cycle-free)
│   │   │   └── timing.go             # Per-layer timing accumulator
│   │   ├── resilience/
│   │   │   ├── breaker.go             # Circuit breaker (in-process + optional Valkey state)
│   │   │   ├── retry.go              # Retry with exponential backoff + jitter
│   │   │   ├── ssrf.go               # SSRF guard: block private/loopback/link-local
│   │   │   └── executor.go           # Composed policy stack: rate-limit → breaker → timeout → retry
│   │   ├── cache/
│   │   │   ├── cache.go              # Cache interface + fail-open wrapper
│   │   │   ├── tiered.go             # L1 (LRU) → L2 (Valkey + breaker) → DB
│   │   │   ├── memory.go             # In-process LRU with TTL
│   │   │   ├── valkey.go             # Valkey backend
│   │   │   └── jitter.go             # TTL jitter ±% (cache avalanche defense)
│   │   ├── pagination/
│   │   │   └── pagination.go         # Page/size extraction, clamping, overflow guard
│   │   └── crypto/
│   │       └── crypto.go             # AES-256-GCM field encryption (optional)
│   │
│   ├── auth/                           # Authentication domain
│   │   ├── service.go                 # Login, token issue/verify, API key resolve
│   │   ├── handler.go                 # POST /auth/login, POST /auth/refresh
│   │   ├── repository.go             # User + API key DB access
│   │   ├── model.go                   # User, Role, Permission, APIKey
│   │   ├── password.go               # bcrypt hash/verify
│   │   ├── jwt.go                     # Mint/verify access + refresh tokens, blacklist
│   │   ├── apikey.go                  # Generate, hash (HMAC-SHA256), verify
│   │   └── rbac.go                    # Permission check: user → roles → permissions
│   │
│   ├── items/                          # Example CRUD domain (deletable by adopters)
│   │   ├── handler.go                 # HTTP endpoints + DTOs + route registration
│   │   ├── service.go                 # Business logic + cache-aside + hooks
│   │   └── repository.go             # Item-specific queries (extends base repo)
│   │
│   ├── store/                          # Database layer
│   │   ├── pool.go                    # pgxpool init, ping, per-call timeout
│   │   ├── repository.go             # Generic Repository[T]: CRUD, list, soft-delete, pagination
│   │   ├── migrate.go                # goose migration runner
│   │   ├── db/                        # sqlc-generated code
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   ├── items.sql.go
│   │   │   └── auth.sql.go
│   │   └── queries/                   # Raw SQL (input to sqlc)
│   │       ├── items.sql
│   │       └── auth.sql
│   │
│   └── valkey/
│       └── client.go                  # go-redis client factory (nil when disabled)
│
├── migrations/
│   ├── 0001_base.sql                  # items + users + roles + permissions + api_keys
│   └── ...
│
├── docs/
│   ├── PRD.md                         # This file
│   ├── ROADMAP.md                     # Phased build plan
│   └── adr/                           # Architecture Decision Records
│       ├── 0001-router-choice.md
│       ├── 0002-response-envelope.md
│       ├── 0003-error-handling.md
│       ├── 0004-resilience-scope.md
│       └── 0005-db-driver-query-gen.md
│
├── loadtest/                           # k6 scripts (mirror busy-api pattern)
│   ├── smoke.js
│   └── load.js
│
├── .github/workflows/
│   └── test.yml                       # lint + test + race detector
│
├── .air.toml                          # Hot reload config
├── .env.example                       # All env vars documented
├── .golangci.yml                      # Linter config
├── .githooks/pre-commit               # gofmt + go vet on staged .go files
├── docker-compose.yml                 # Postgres + Valkey + app
├── Dockerfile                         # Multi-stage: build → distroless, non-root
├── Makefile                           # Dev targets
├── sqlc.yaml                          # sqlc config
├── go.mod / go.sum
├── CLAUDE.md
└── README.md
```

**Key structural decision:** `internal/platform/` holds all domain-free infrastructure (response envelope, errors, middleware, resilience, cache, logging). Domain packages (`auth/`, `items/`) import platform — never the reverse. This mirrors:
- Django: `apps/core/` (infrastructure) vs `apps/accounts/`, `apps/items/` (domain)
- FastAPI: `src/core/` vs `src/api/`, `src/service/`
- document-service: `internal/platform/` vs `internal/entity/`, `internal/docservice/`

---

## 6. Contracts (What Must Match Across the Trio)

These are the shared opinions. Divergence here would break the "same doctrine, three runtimes" story.

### 6.1 Response Envelope

Every HTTP response uses one shape:

```json
{
  "success": true,
  "message": "Item created",
  "data": { "id": 1, "name": "...", "created_at": "..." },
  "errors": null,
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Error shape:

```json
{
  "success": false,
  "message": "Validation failed",
  "data": null,
  "errors": [
    { "code": "VALIDATION_ERROR", "message": "name is required", "field": "name" }
  ],
  "request_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Paginated shape wraps `data`:

```json
{
  "success": true,
  "message": "Items retrieved",
  "data": {
    "items": [...],
    "page": 1,
    "size": 20,
    "total_count": 150,
    "total_pages": 8,
    "has_prev": false,
    "has_next": true
  },
  "request_id": "..."
}
```

### 6.2 Typed Error Codes

Stable, machine-readable, UPPER_SNAKE_CASE. Consumers build switch statements against these.

| Code | HTTP Status | When |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Input fails validation |
| `INVALID_INPUT` | 400 | Malformed request body / params |
| `UNAUTHORIZED` | 401 | Missing or invalid credentials |
| `FORBIDDEN` | 403 | Authenticated but lacks permission |
| `NOT_FOUND` | 404 | Entity does not exist or is soft-deleted |
| `CONFLICT` | 409 | Duplicate key, stale version, inactive parent |
| `PAYLOAD_TOO_LARGE` | 413 | Request body exceeds limit |
| `RATE_LIMITED` | 429 | Rate limit exceeded |
| `INTERNAL_ERROR` | 500 | Unhandled error (sanitized message in prod) |
| `EXTERNAL_ERROR` | 502 | Upstream returned 4xx (not 429) |
| `TRANSIENT_ERROR` | 502 | Upstream 5xx / timeout (trips breaker) |
| `SERVICE_UNAVAILABLE` | 503 | Circuit breaker open / dependency down |

### 6.3 Layering

```
HTTP Request
  → Middleware (recovery, request-id, logging, CORS, security, body-limit, rate-limit, auth)
    → Handler (validate input, call service, return envelope)
      → Service (business logic, hooks, cache-aside, transactions)
        → Repository (DB queries via sqlc, generic CRUD)
          → pgxpool (connection pool)
```

**Rules** (same as Python boilerplates):
- Handlers never touch DB directly; only call services
- Services never construct HTTP responses; return data or error
- Repositories never import handler or service packages
- Platform packages never import domain packages

### 6.4 Base Model Fields

Every table:

| Column | Type | Default | Purpose |
|---|---|---|---|
| `id` | `bigserial` | auto | Primary key |
| `created_at` | `timestamptz` | `now()` | Row creation |
| `updated_at` | `timestamptz` | `now()` | Last modification |
| `is_active` | `boolean` | `true` | Soft-delete flag |

Domain tables add their own columns on top.

### 6.5 Soft Delete

Default: `is_active = false`. All list queries filter `is_active = true` unless caller opts out. Hard delete available but not the default path.

### 6.6 Request-ID Propagation

1. Middleware mints UUIDv4 or adopts valid `X-Request-ID` header
2. Stored in `context.Context`
3. Injected into every slog log line
4. Returned in response envelope
5. Echoed in `X-Request-ID` response header

### 6.7 Health Probes

| Endpoint | Check | Use |
|---|---|---|
| `GET /healthz` | Always 200 (liveness) | Load balancer |
| `GET /readyz` | DB ping + Valkey ping (readiness) | Orchestrator |

Readiness returns `{"ready": true, "checks": {"postgres": "ok", "valkey": "ok"}}`. Soft dependency (Valkey) degrades to "degraded" but doesn't fail readiness.

---

## 7. Middleware Stack (Ordered)

Same order as busy-api + document-service, which aligns with Django/FastAPI boilerplates:

| Order | Middleware | Purpose |
|---|---|---|
| 1 | Recovery | Catch panics → 500 envelope |
| 2 | BodyLimit | MaxBytesReader → 413 |
| 3 | CORS | Configurable allowlist |
| 4 | SecurityHeaders | CSP, HSTS, X-Frame-Options, X-Content-Type-Options |
| 5 | RequestID | Mint/adopt UUID, echo in response |
| 6 | RequestLogging | Structured access log, per-layer timing, Server-Timing header |
| 7 | RateLimit | IP-keyed (public) / entity-keyed (protected) |
| 8 | Auth | JWT / API key → entity context (protected routes only) |

---

## 8. Auth

### 8.1 JWT (Primary)

- Access token: short-lived (15 min default), signed HMAC-SHA256
- Refresh token: longer-lived (7 days default), same signing
- Blacklist: refresh token `jti` stored in Valkey on logout/rotation
- Blacklist fail-open on Valkey outage for access tokens (short-lived), fail-closed for refresh

### 8.2 API Key

- Random 32-byte hex, shown once on creation
- Stored as HMAC-SHA256 hash (with pepper) — pattern from document-service
- Prefix (first 8 chars) used for DB lookup, then constant-time hash compare
- Sent via `X-API-Key` header

### 8.3 RBAC

- User → M2M → Role → M2M → Permission
- Permission = `(resource, action)` pair (e.g., `items:create`, `items:read`)
- Superuser role short-circuits all checks
- Per-request permission cache on context

### 8.4 Route Groups

| Group | Auth | Rate Limit |
|---|---|---|
| Public | None | IP-keyed |
| Protected | JWT or API key required | Entity-keyed |
| Admin | Superuser role required | Entity-keyed |

---

## 9. Resilience

### 9.1 Circuit Breaker

States: CLOSED → OPEN → HALF_OPEN → CLOSED.

**In-process** (memory state) for v0.1.0. Same pattern as busy-api's `internal/resilience/breaker/`. Two-tier (Valkey primary + memory fallback) is the document-service pattern — evaluate for v0.2.

Config: `BREAKER_FAIL_THRESHOLD` (5), `BREAKER_RECOVERY_S` (30).

Applied to: outbound HTTP calls, Valkey cache layer.

### 9.2 Retry

Exponential backoff with full jitter: `base * 2^attempt + random(0, base)`.

Config: `RETRY_MAX` (3), `RETRY_BASE_MS` (200).

Only retries transient errors (5xx, timeouts, network). Never retries 4xx.

### 9.3 SSRF Guard

Block private, loopback, link-local, multicast IPs. Validate at boot for config URLs, per-request for dynamic URLs.

Same implementation as document-service's `resilience.AssertPublicURL`.

### 9.4 Cache Stack

```
L1 (in-process LRU, 10k entries, 30s TTL)
  ↓ miss
L2 (Valkey, breaker-guarded, TTL with ±10% jitter)
  ↓ miss or breaker OPEN
Database
```

Defenses (all proven in busy-api):
- **Singleflight**: collapse concurrent cache misses on same key
- **Negative cache**: tombstone for confirmed-absent IDs (30s TTL)
- **Bloom filter**: optional, pre-filter for cache penetration DoS
- **TTL jitter**: ±% spread prevents synchronized expiry
- **Fail-open**: cache errors → treated as miss, never 5xx

### 9.5 Rate Limiting

Inbound: fixed-window counter per IP (public) or per entity (protected).

Backend: Valkey primary, in-memory fallback (same two-tier pattern as document-service).

Config: `RATE_LIMIT_RPM` (120), `LOGIN_RATE_LIMIT_RPM` (10).

---

## 10. Configuration

Env-based via `caarlos0/env`. `.env` file for local dev, AWS Secrets Manager overlay for prod.

### 10.1 Required

| Var | Purpose |
|---|---|
| `DATABASE_URL` | Postgres DSN |
| `AUTH_TOKEN_SECRET` | JWT signing key (HMAC-SHA256) |
| `API_KEY_HASH_PEPPER` | HMAC pepper for API key storage |

### 10.2 Optional (with defaults)

| Var | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `ENV` | `local` | Environment (local/dev/staging/prod) |
| `DB_MAX_CONNS` | `0` (4×CPU) | pgxpool max connections |
| `DB_QUERY_TIMEOUT_MS` | `2000` | Per-call query timeout |
| `VALKEY_URL` | `redis://localhost:6379/0` | Cache backend (empty = disabled) |
| `CACHE_ITEM_TTL_S` | `300` | L2 cache TTL |
| `CACHE_L1_ENABLED` | `true` | In-process LRU |
| `CACHE_L1_MAX` | `10000` | L1 entry limit |
| `CACHE_TTL_JITTER_PCT` | `10` | L2 TTL ±% spread |
| `LOG_LEVEL` | `INFO` | slog level |
| `LOG_JSON` | `true` | JSON vs text output |
| `MAX_BODY_BYTES` | `1048576` | Request body limit (1MB) |
| `CORS_ORIGINS` | `` | Comma-separated allowlist |
| `BREAKER_FAIL_THRESHOLD` | `5` | Breaker opens after N failures |
| `BREAKER_RECOVERY_S` | `30` | OPEN → HALF_OPEN interval |
| `RETRY_MAX` | `3` | Retry attempts |
| `RETRY_BASE_MS` | `200` | Backoff base |
| `RATE_LIMIT_RPM` | `120` | Requests per minute |
| `JWT_ACCESS_TTL_M` | `15` | Access token TTL (minutes) |
| `JWT_REFRESH_TTL_H` | `168` | Refresh token TTL (hours, 7 days) |
| `SECRETS_MANAGER_SECRET_ID` | `` | AWS Secrets Manager bundle (empty = skip) |

---

## 11. Database Schema (v0.1.0)

### 11.1 Core Tables

```sql
-- Users
CREATE TABLE users (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    updated_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    name            varchar(255)    NOT NULL,
    email           varchar(255)    NOT NULL UNIQUE,
    password_hash   text            NOT NULL,
    is_superuser    boolean         NOT NULL DEFAULT false
);

-- Roles
CREATE TABLE roles (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    updated_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    name            varchar(100)    NOT NULL UNIQUE,
    code            varchar(100)    NOT NULL UNIQUE
);

-- Permissions
CREATE TABLE permissions (
    id              bigserial       PRIMARY KEY,
    resource        varchar(100)    NOT NULL,
    action          varchar(50)     NOT NULL,
    UNIQUE (resource, action)
);

-- User ↔ Role
CREATE TABLE user_roles (
    user_id         bigint          NOT NULL REFERENCES users(id),
    role_id         bigint          NOT NULL REFERENCES roles(id),
    PRIMARY KEY (user_id, role_id)
);

-- Role ↔ Permission
CREATE TABLE role_permissions (
    role_id         bigint          NOT NULL REFERENCES roles(id),
    permission_id   bigint          NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

-- API Keys
CREATE TABLE api_keys (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    user_id         bigint          NOT NULL REFERENCES users(id),
    name            varchar(255)    NOT NULL,
    prefix          varchar(8)      NOT NULL,
    key_hash        text            NOT NULL,
    last_used_at    timestamptz
);
CREATE INDEX idx_api_keys_prefix ON api_keys(prefix) WHERE is_active = true;
```

### 11.2 Example Domain Table

```sql
-- Items (example CRUD resource — delete when forking)
CREATE TABLE items (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    updated_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    name            varchar(255)    NOT NULL,
    code            varchar(100)    NOT NULL UNIQUE,
    notes           jsonb
);
CREATE INDEX idx_items_active_id ON items(id) WHERE is_active = true;
```

---

## 12. API Surface (v0.1.0)

### 12.1 Infrastructure

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/healthz` | None | Liveness probe |
| GET | `/readyz` | None | Readiness probe |

### 12.2 Auth

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/auth/login` | None | Email + password → access + refresh tokens |
| POST | `/api/v1/auth/refresh` | None | Refresh token → new access + refresh |
| POST | `/api/v1/auth/logout` | Protected | Blacklist current refresh token |

### 12.3 Items (Example Resource)

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/items` | Protected | Create item |
| GET | `/api/v1/items` | Protected | List items (paginated) |
| GET | `/api/v1/items/:id` | Protected | Get item by ID |
| PATCH | `/api/v1/items/:id` | Protected | Update item |
| DELETE | `/api/v1/items/:id` | Protected | Soft-delete item |

### 12.4 Admin

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/admin/users` | Admin | Create user |
| GET | `/api/v1/admin/users` | Admin | List users |
| POST | `/api/v1/admin/users/:id/api-keys` | Admin | Issue API key |
| POST | `/api/v1/admin/roles` | Admin | Create role |
| POST | `/api/v1/admin/roles/:id/permissions` | Admin | Assign permissions |

---

## 13. Docker

### 13.1 Dockerfile

Multi-stage build:
1. **Builder**: `golang:1.22-alpine`, compile with `CGO_ENABLED=0`
2. **Runtime**: `gcr.io/distroless/static-debian12`, non-root user, copy binary only

### 13.2 docker-compose.yml

| Service | Image | Purpose | Healthcheck |
|---|---|---|---|
| `postgres` | `postgres:16-alpine` | Primary DB | `pg_isready` |
| `valkey` | `valkey/valkey:8-alpine` | Cache + rate-limit + JWT blacklist | `valkey-cli ping` |
| `app` | Build from Dockerfile | API server | `curl /healthz` |
| `migrate` | Build from Dockerfile | One-shot migration runner | exits 0 |

---

## 14. Makefile Targets

| Target | Purpose |
|---|---|
| `make dev` | Hot reload via air |
| `make run` | Build + run once |
| `make build` | Compile to `bin/server` |
| `make test` | `go test -race ./...` |
| `make lint` | `golangci-lint run` |
| `make fmt` | `gofumpt -w .` |
| `make migrate-up` | Run pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create` | Create new migration file |
| `make sqlc` | Regenerate sqlc code |
| `make compose-up` | Start Postgres + Valkey |
| `make compose-down` | Stop stack |
| `make hooks` | Install git hooks |
| `make load-smoke` | k6 smoke test |
| `make load` | k6 load test |
| `make swagger` | Generate OpenAPI via swag |

---

## 15. CI Pipeline (.github/workflows/test.yml)

1. `golangci-lint run`
2. `go vet ./...`
3. `go test -race -coverprofile=coverage.out ./...`
4. Coverage gate (target: 70% overall)
5. Build binary (verify compilation)

Services: Postgres 16 + Valkey 8 (for integration tests).

---

## 16. Phased Build Plan

### M0: Foundation (Weekend 1)

- [ ] `go mod init`, Makefile, .env.example, .air.toml, .golangci.yml
- [ ] `internal/config/` — env-based config struct
- [ ] `internal/platform/logger/` — slog setup, JSON, request-id context handler
- [ ] `internal/platform/reqcontext/` — request-ID carrier, per-layer timing
- [ ] `internal/platform/apperr/` — AppError type, stable codes, HTTP status registry
- [ ] `internal/platform/response/` — envelope: Success, Error, Paginated
- [ ] `internal/platform/middleware/` — full stack (recovery, body-limit, CORS, security-headers, request-id, request-logging)
- [ ] `internal/platform/httpserver/` — server, router (public/protected/admin groups), probes
- [ ] `cmd/server/main.go` — wire config + logger + router, graceful shutdown
- [ ] docker-compose.yml (Postgres + Valkey)
- [ ] Dockerfile (multi-stage distroless)
- [ ] Verify: `docker compose up`, `curl /healthz` → 200, `curl /readyz` → 200
- [ ] ADR-0001: Router choice (Gin)
- [ ] ADR-0002: Response envelope shape

### M1: Database + Example CRUD (Weekend 1-2)

- [ ] `internal/store/` — pgxpool init, per-call timeout, Atomic txn wrapper
- [ ] `internal/valkey/` — go-redis client factory
- [ ] Migration 0001: items table
- [ ] `sqlc.yaml` + `internal/store/queries/items.sql` + generate
- [ ] `internal/store/repository.go` — Generic Repository[T]: CRUD, list, soft-delete, pagination
- [ ] `internal/items/` — handler, service, repository (full CRUD round-trip)
- [ ] Verify: POST/GET/LIST/DELETE /api/v1/items works end-to-end
- [ ] ADR-0005: DB driver + query gen choice (pgx + sqlc)

### M2: Auth (Weekend 2)

- [ ] Migration 0002: users, roles, permissions, api_keys, junction tables
- [ ] `internal/auth/` — JWT (issue/verify/blacklist), API key (generate/hash/verify), password (bcrypt), RBAC
- [ ] `internal/platform/middleware/auth.go` — JWT + API key resolver
- [ ] `internal/platform/middleware/rate_limit.go` — IP + entity-keyed
- [ ] Auth endpoints: login, refresh, logout
- [ ] Admin endpoints: user CRUD, API key issue, role/permission management
- [ ] Protect items routes behind auth middleware
- [ ] Verify: full auth flow (login → access token → CRUD → refresh → logout)

### M3: Resilience + Cache (Weekend 3)

- [ ] `internal/platform/resilience/` — breaker, retry, SSRF guard, executor
- [ ] `internal/platform/cache/` — tiered cache (L1 LRU → L2 Valkey + breaker → DB)
- [ ] Wire cache-aside into items service (get + list)
- [ ] Singleflight, negative cache, TTL jitter
- [ ] Verify: kill Valkey mid-request → fail-open, items still served from DB
- [ ] ADR-0004: Resilience scope

### M4: Polish (Weekend 3-4)

- [ ] `internal/config/secrets.go` — AWS Secrets Manager overlay
- [ ] OpenAPI via swaggo/swag annotations on all handlers
- [ ] .githooks/pre-commit (gofmt + go vet)
- [ ] .github/workflows/test.yml — CI pipeline
- [ ] k6 load test scripts (smoke + load)
- [ ] CLAUDE.md
- [ ] README.md (first-time-user flow)
- [ ] ADR-0003: Error handling design
- [ ] Tag v0.1.0

---

## 17. Out of Scope (v0.1.0)

- gRPC
- WebSocket / SSE
- Background jobs (Asynq/River)
- Server-rendered HTML
- OAuth2 providers (Google etc.) — JWT + API key only for v0.1
- Observability stack (OpenTelemetry, Prometheus, Grafana) — structured logs only
- Two-tier breaker state (Valkey + memory) — in-process only
- Bloom filter — optional, not wired by default
- Audit log table — structured logs serve this role for v0.1
- Field encryption (AES-256-GCM) — defer until a real consumer needs it

---

## 18. Definition of Done (v0.1.0)

1. `docker compose up -d` brings up Postgres + Valkey + app
2. `curl /healthz` → 200
3. `curl /readyz` → 200 with postgres + valkey checks
4. Full auth flow: login → access token → CRUD items → refresh → logout
5. Every response carries the typed envelope with request_id
6. Every error returns a stable error code from the registry
7. Request-ID appears in every log line and every response
8. `go test -race ./...` passes clean
9. `golangci-lint run` passes clean
10. README documents: clone → `.env` → `docker compose up` → curl works
11. One ADR per major decision (5 minimum)
12. Tagged `v0.1.0`
