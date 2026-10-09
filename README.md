# gin_boilerplate

[![CI](https://github.com/prajwalmahajan101/gin_boilerplate/actions/workflows/ci.yml/badge.svg)](https://github.com/prajwalmahajan101/gin_boilerplate/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/prajwalmahajan101/gin_boilerplate?sort=semver)](https://github.com/prajwalmahajan101/gin_boilerplate/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/prajwalmahajan101/gin_boilerplate)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/prajwalmahajan101/gin_boilerplate)](https://goreportcard.com/report/github.com/prajwalmahajan101/gin_boilerplate)
[![License: MIT](https://img.shields.io/github/license/prajwalmahajan101/gin_boilerplate)](LICENSE)

Production-shaped Gin REST starter. Third leg of the boilerplate trio (Django / FastAPI / Gin).

**Status:** v0.1.0 -- first release. In-process resilience; distributed/two-tier is v0.2 (see [ADR 0004](docs/adr/0004-resilience-scope.md)).

## What This Is

Forkable Go service skeleton with the same opinions as [django_boilerplate](https://github.com/prajwalch/django_boilerplate) and [fastapi_boilerplate](https://github.com/prajwalch/fastapi_boilerplate):

- Typed errors with stable error codes
- Unified JSON response envelope
- Request-ID correlation end-to-end
- Handler -> Service -> Repository layering
- BaseService with pre/post hooks and soft-delete cascade
- Modular monolith architecture
- JWT + API key auth with RBAC
- Tiered cache (L1 LRU -> L2 Valkey -> DB)
- Circuit breaker, retry, SSRF guard
- Structured logging (slog)
- OpenAPI via Swagger
- Docker, CI, GoReleaser

## Stack

Go 1.26 / Gin / pgx + sqlc / golang-migrate / slog / golang-jwt / go-redis / golangci-lint / k6 / Docker (distroless) / GoReleaser

## Quick Start

```bash
git clone https://github.com/prajwalmahajan101/gin_boilerplate.git
cd gin_boilerplate
cp .env.example .env           # fill in required values (set AUTH_TOKEN_SECRET)
docker compose up -d           # postgres + valkey
make migrate-up                # run migrations
make dev                       # hot-reload server on :8080
```

Check it's alive and open the API docs:

```bash
curl localhost:8080/healthz    # liveness
curl localhost:8080/readyz     # readiness (postgres + valkey)
open  localhost:8080/swagger/index.html  # interactive API docs
```

### First request: register → token → CRUD

All `/api/v1/items` routes require a Bearer token. Register a user, grab the
access token from the envelope, and use it:

```bash
BASE=localhost:8080/api/v1

# 1. Register — the response envelope's data carries the JWT pair
TOKEN=$(curl -s -X POST $BASE/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"password123"}' \
  | jq -r .data.access_token)

# 2. Create an item
curl -s -X POST $BASE/items \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"first item","code":"ITEM-001"}' | jq

# 3. List items
curl -s $BASE/items -H "Authorization: Bearer $TOKEN" | jq
```

Every response is the same envelope: `{success, message, data, errors, request_id}`.
Already have a user? `POST /api/v1/auth/login` returns the same token pair.

### Load tests

```bash
make load-smoke                # 1 VU, full journey — quick sanity (server must be up)
make load VUS=50 DURATION=2m   # ramped load (raise RATE_LIMIT_RPM on the server)
```

## Development

```bash
make build      # compile to bin/server
make test       # go test -race ./...
make lint       # golangci-lint (--build-tags integration)
make fmt        # gofmt -w .
make vet        # go vet (-tags integration)
make check      # vet + lint + test (full verification gate)
make vuln       # govulncheck
make hooks      # install pre-commit hooks
make swagger    # regenerate OpenAPI spec
make sqlc       # regenerate query code
```

## Project Structure

```
cmd/server/main.go              entry point
internal/
  platform/                     cross-cutting infrastructure
    apperr/                     typed errors + code registry
    response/                   JSON envelope
    middleware/                  HTTP middleware stack
    httpserver/                  server lifecycle, router, probes
    logger/                     slog + request-id injection
    cache/                      tiered cache
    resilience/                 breaker, retry, SSRF
  store/                        generic persistence
    basemodel.go                shared model fields
    repository.go               Repository[T] generic CRUD
    service.go                  BaseService[T] with hooks
  modules/                      domain modules
    items/                      example CRUD (delete when forking)
    auth/                       JWT, API key, RBAC
  valkey/                       go-redis client
  config/                       env-based config
```

## Docs

- [PRD](docs/PRD.md) -- requirements, contracts, schema
- [Roadmap](docs/ROADMAP.md) -- extraction plan with source references
- [ADRs](docs/adr/) -- architecture decision records

## License

MIT
