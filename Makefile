.PHONY: help up down dev watch build vet \
        migrate-up migrate-down migrate-force migrate-drop \
        sqlc \
        test test-race test-cover test-integration test-e2e \
        install-hooks \
        cache-flush clean clean-all

E2E_BASE_URL ?= http://localhost:3000

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Git hooks ─────────────────────────────────────────────────────────────────

install-hooks: ## Install git hooks (run once after cloning)
	cp scripts/hooks/pre-commit .git/hooks/pre-commit
	cp scripts/hooks/pre-push   .git/hooks/pre-push
	cp scripts/hooks/commit-msg .git/hooks/commit-msg
	chmod +x .git/hooks/pre-commit .git/hooks/pre-push .git/hooks/commit-msg
	@echo "Git hooks installed."

# ── Infrastructure ────────────────────────────────────────────────────────────

up: ## Start Postgres + pgBouncer + Redis
	docker compose up -d

down: ## Stop and remove containers
	docker compose down

# ── Application ───────────────────────────────────────────────────────────────

dev: ## Run the API server (no hot reload)
	go run ./cmd/api

watch: ## Run the API server with hot reload (requires: go install github.com/air-verse/air@latest)
	air

build: ## Compile all packages
	go build ./...

vet: ## Run go vet
	go vet ./...

# ── Migrations ────────────────────────────────────────────────────────────────

migrate-up: ## Apply all pending migrations
	go run ./cmd/migrate up

migrate-down: ## Roll back one migration
	go run ./cmd/migrate down

migrate-force: ## Force migration version  (usage: make migrate-force V=3)
	go run ./cmd/migrate force $(V)

migrate-drop: ## Drop entire schema (blocked in production)
	go run ./cmd/migrate drop

# ── Code generation ───────────────────────────────────────────────────────────

sqlc: ## Regenerate sqlc code from sql/
	sqlc generate

# ── Tests ─────────────────────────────────────────────────────────────────────

test: ## Unit tests (no infrastructure)
	go test ./...

test-race: ## Unit tests with race detector
	go test -race ./...

test-cover: ## Unit tests + HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

test-integration: ## Integration tests (requires Postgres — run `make up` first)
	go test -tags integration ./internal/repository/...

test-e2e: ## E2E tests (requires full stack — run `make up dev` first)
	E2E_BASE_URL=$(E2E_BASE_URL) go test -tags e2e ./internal/e2e/...

# ── Cleanup ───────────────────────────────────────────────────────────────────

cache-flush: ## Flush all Redis keys (requires `make up`)
	docker compose exec redis redis-cli FLUSHALL

clean: ## Remove coverage output + Go test cache
	rm -f coverage.out
	go clean -testcache

clean-all: ## clean + Go build cache + Redis flush (requires `make up`)
	rm -f coverage.out
	go clean -testcache
	go clean -cache
	docker compose exec redis redis-cli FLUSHALL
