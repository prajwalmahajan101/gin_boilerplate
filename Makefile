BINARY := bin/server
MAIN   := ./cmd/server

.DEFAULT_GOAL := help

.PHONY: help build run dev test lint fmt vet hooks vuln \
        migrate-up migrate-down migrate-create sqlc swagger \
        compose-up compose-down seed load-smoke load

help: ## Show this help message
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'
	@echo ""

build: ## Compile the binary to bin/server
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 go build -o $(BINARY) $(MAIN)

run: build ## Build and run the server
	./$(BINARY)

dev: ## Start development server with hot-reload (requires air)
	air

test: ## Run all tests with race detector
	go test -race -timeout 5m ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code with gofmt
	gofmt -w .

vet: ## Run go vet
	go vet ./...

hooks: ## Install pre-commit hooks (gofmt + vet/lint with -tags integration; govulncheck on push)
	git config --unset core.hooksPath 2>/dev/null || true
	pre-commit install --hook-type pre-commit --hook-type pre-push
	@echo "pre-commit hooks installed. See .pre-commit-config.yaml"

vuln: ## Run govulncheck
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

migrate-up: ## Apply pending migrations
	go run ./cmd/migrate up

migrate-down: ## Roll back migrations (override steps: make migrate-down n=2)
	go run ./cmd/migrate down $(or $(n),1)

migrate-create: ## Create a new migration (make migrate-create name=add_foo)
	go run ./cmd/migrate create $(name)

sqlc: ## Regenerate sqlc query code
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.30.0 generate

swagger: ## Regenerate OpenAPI spec
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/server/main.go -o docs/swagger

seed: ## Seed admin user (requires SEED_ADMIN_PASSWORD env)
	go run ./cmd/seed

compose-up: ## Start docker-compose stack
	docker compose up -d

compose-down: ## Stop docker-compose stack
	docker compose down

load-smoke: ## Run k6 smoke test (stub -- wired in M10)
	@echo "stub: not yet wired (M10)"

load: ## Run k6 load test (stub -- wired in M10)
	@echo "stub: not yet wired (M10)"
