# Contributing

Thanks for your interest in `gin_boilerplate`. This is a forkable starter, so
contributions that keep it small, opinionated, and production-shaped are most
welcome — bug fixes, docs, and focused improvements over large new surfaces.

## Getting started

```bash
git clone https://github.com/prajwalmahajan101/gin_boilerplate.git
cd gin_boilerplate
cp .env.example .env        # set AUTH_TOKEN_SECRET at minimum
docker compose up -d        # postgres + valkey
make migrate-up
make dev                    # hot-reload server on :8080
make hooks                  # install pre-commit hooks (recommended)
```

See [`CLAUDE.md`](CLAUDE.md) for the full command list, project layout, and
conventions, and [`docs/adr/`](docs/adr/) for the decisions behind the design.

## Workflow

1. **Branch** off `main` — never commit to `main` directly. Use a descriptive
   name, e.g. `feat/<topic>`, `fix/<topic>`, `docs/<topic>`.
2. **Make the change.** Match the surrounding code; keep handlers thin, business
   logic in services, persistence in repositories.
3. **Verify locally** before opening a PR:
   ```bash
   make check      # go vet + golangci-lint + go test -race ./... (all -tags integration)
   make vuln       # govulncheck
   ```
4. **Open a PR** into `main`. CI must be green before merge (build/vet/test,
   golangci-lint, govulncheck, integration tests, Docker build).

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<optional scope>): <subject>
```

- Types: `feat`, `fix`, `refactor`, `docs`, `test`, `chore`.
- Subject in the imperative mood, ≤72 chars, no trailing period.
- Keep commits **atomic** — one logical change each.

Example: `feat(auth): add forgot/reset-password flow over SES`

## Code style & checks

- **Format:** `make fmt` (gofmt).
- **Lint:** `make lint` — golangci-lint, run with `--build-tags integration`.
- **Tests:** prefer integration/e2e over heavily-mocked unit tests. Integration
  tests use the `//go:build integration` tag; `make check` compiles and runs
  them.
- **Errors:** return typed `apperr.*` errors with stable codes from service
  boundaries; never a bare `errors.New`. Handlers render via `response.Error`.
- **Config:** add new env vars to `internal/config/config.go` (with an
  `envDefault`) and mirror them in `.env.example`; redact secrets in `LogValue`.
- **Generated code:** regenerate and commit after changes —
  `make sqlc` (queries), `make swagger` (OpenAPI).

## Adding a domain module

Mirror `internal/modules/items`: a handler + a service (compose
`BaseService[T]`) + queries, implementing
`RegisterRoutes(public, protected, admin)`, then wire it in `cmd/server/main.go`.
Depend on a small service interface declared in the handler so it stays
fake-testable.

## Architecture decisions

Non-trivial design changes (new dependency, protocol change, schema migration)
should reference or add an ADR in [`docs/adr/`](docs/adr/) using the
Context / Decision / Consequences / Usage template. Update
[`docs/adr/INDEX.md`](docs/adr/INDEX.md) and the ADR index in `CLAUDE.md`.

## Reporting issues

Open a GitHub issue with a clear title, what you expected vs. what happened, and
a minimal reproduction (request + response envelope, logs with the `request_id`)
where possible.

## License

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).
