BINARY := bin/server
MAIN   := ./cmd/server

.DEFAULT_GOAL := help

.PHONY: help build run dev test lint fmt vet hooks vuln \
        migrate-up migrate-down migrate-create sqlc swagger \
        compose-up compose-down load-smoke load

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

hooks: ## Install git pre-commit hooks
	git config core.hooksPath .githooks
	chmod +x .githooks/pre-commit
	@echo "pre-commit hook installed (gofmt + go vet on staged .go)."

vuln: ## Run govulncheck
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

migrate-up: ## Apply pending migrations (stub -- wired in M4)
	@echo "stub: not yet wired (M4)"

migrate-down: ## Roll back last migration (stub -- wired in M4)
	@echo "stub: not yet wired (M4)"

migrate-create: ## Create a new migration (stub -- wired in M4)
	@echo "stub: not yet wired (M4)"

sqlc: ## Regenerate sqlc query code (stub -- wired in M4)
	@echo "stub: not yet wired (M4)"

swagger: ## Regenerate OpenAPI spec (stub -- wired in M3)
	@echo "stub: not yet wired (M3)"

compose-up: ## Start docker-compose stack
	docker compose up -d

compose-down: ## Stop docker-compose stack
	docker compose down

load-smoke: ## Run k6 smoke test (stub -- wired in M10)
	@echo "stub: not yet wired (M10)"

load: ## Run k6 load test (stub -- wired in M10)
	@echo "stub: not yet wired (M10)"
