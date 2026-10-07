# gin_boilerplate

Production-shaped Gin REST starter. Third leg of the boilerplate trio (Django / FastAPI / Gin).

**Status:** In Progress -- building milestone by milestone.

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

Go 1.22+ / Gin / pgx + sqlc / golang-migrate / slog / golang-jwt / go-redis / golangci-lint / Docker (distroless) / GoReleaser

## Quick Start

```bash
git clone https://github.com/prajwalch/gin_boilerplate.git
cd gin_boilerplate
cp .env.example .env          # fill in required values
docker compose up -d           # postgres + valkey
make migrate-up                # run migrations
make dev                       # hot-reload server on :8080
```

```bash
curl localhost:8080/healthz    # liveness
curl localhost:8080/readyz     # readiness (postgres + valkey)
curl localhost:8080/swagger/index.html  # API docs
```

## Development

```bash
make build      # compile to bin/server
make test       # go test -race ./...
make lint       # golangci-lint
make fmt        # gofumpt
make vet        # go vet
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
