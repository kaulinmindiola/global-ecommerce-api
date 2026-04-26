# ============================================================================
# GLOBAL E-COMMERCE API - MAKEFILE
# ============================================================================
# Professional build automation for development workflow.
# Run `make help` to see all available commands.
# ============================================================================

# ─── Variables ──────────────────────────────────────────────────────────────
APP_NAME      := global-ecommerce-api
BINARY_NAME   := ecommerce-api
BINARY_PATH   := ./bin/$(BINARY_NAME)
MAIN_PACKAGE  := ./cmd/api

# Build metadata injected at compile time
VERSION       := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -ldflags "\
	-w -s \
	-X main.version=$(VERSION) \
	-X main.buildDate=$(BUILD_DATE) \
	-X main.commitHash=$(COMMIT_HASH)"

# Colors for output
COLOR_RESET  := \033[0m
COLOR_BOLD   := \033[1m
COLOR_GREEN  := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE   := \033[34m
COLOR_CYAN   := \033[36m

# Files
GO_FILES      := $(shell find . -type f -name '*.go' -not -path "./vendor/*")
COVERAGE_FILE := coverage.txt
COVERAGE_HTML := coverage.html

.DEFAULT_GOAL := help

## ─── Help ───────────────────────────────────────────────────────────────────
## help: Display this help message
.PHONY: help
help:
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Global E-commerce API - Available Commands$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  $(COLOR_CYAN)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } \
		/^## ───/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 4) }' $(MAKEFILE_LIST)
	@echo ""

## ─── Setup & Development ────────────────────────────────────────────────────
## setup: Initial project setup (tools, env, deps)
.PHONY: setup
setup:
	@echo "$(COLOR_GREEN)Setting up project...$(COLOR_RESET)"
	@[ -f .env ] || cp .env.example .env && echo "✓ .env created from .env.example"
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@mkdir -p bin logs
	@echo "$(COLOR_GREEN)✓ Setup complete!$(COLOR_RESET)"

## dev: Start infrastructure (PostgreSQL, Redis, Adminer) via Docker Compose
.PHONY: dev
dev:
	@echo "$(COLOR_GREEN)Starting infrastructure...$(COLOR_RESET)"
	@docker compose up -d postgres redis adminer
	@echo "$(COLOR_GREEN)✓ Services started!$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)PostgreSQL: localhost:5432$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Redis: localhost:6379$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Adminer UI: http://localhost:8080$(COLOR_RESET)"

## stop: Stop infrastructure services
.PHONY: stop
stop:
	@echo "$(COLOR_YELLOW)Stopping infrastructure...$(COLOR_RESET)"
	@docker compose down
	@echo "$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)"

## run: Run API locally (without Docker)
.PHONY: run
run:
	@echo "$(COLOR_GREEN)Running API server...$(COLOR_RESET)"
	@go run $(LDFLAGS) $(MAIN_PACKAGE)

## build: Build production binary for Linux AMD64
.PHONY: build
build:
	@echo "$(COLOR_GREEN)Building production binary...$(COLOR_RESET)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "$(COLOR_GREEN)✓ Binary created: $(BINARY_PATH)$(COLOR_RESET)"

## clean: Remove build artifacts and caches
.PHONY: clean
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf bin/ $(COVERAGE_FILE) $(COVERAGE_HTML) api/docs/
	@go clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"


## ─── Code Quality & Testing ─────────────────────────────────────────────────
## format: Format Go code (gofmt & goimports)
.PHONY: format
format:
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w $(GO_FILES)
	@goimports -w $(GO_FILES)
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

## lint: Run linter (golangci-lint)
.PHONY: lint
lint:
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@golangci-lint run --timeout=5m ./...
	@echo "$(COLOR_GREEN)✓ Linting complete$(COLOR_RESET)"

## check: Run all checks (format + vet + lint)
.PHONY: check
check: format lint
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	@go vet ./...
	@echo "$(COLOR_GREEN)✓ All checks passed$(COLOR_RESET)"

## test: Run unit tests with coverage
.PHONY: test
test:
	@echo "$(COLOR_GREEN)Running unit tests...$(COLOR_RESET)"
	@go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./internal/...
	@echo "$(COLOR_GREEN)✓ Tests complete$(COLOR_RESET)"

## test-coverage: Run tests and open HTML coverage report
.PHONY: test-coverage
test-coverage: test
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(COVERAGE_HTML)$(COLOR_RESET)"

## swagger: Generate Swagger API documentation
.PHONY: swagger
swagger:
	@echo "$(COLOR_GREEN)Generating Swagger docs...$(COLOR_RESET)"
	@swag init -g cmd/api/main.go -o api/docs
	@echo "$(COLOR_GREEN)✓ Swagger docs generated$(COLOR_RESET)"


## ─── Database & Redis ───────────────────────────────────────────────────────
## db-create: Create database tables from schema.sql (Cross-platform)
.PHONY: db-create
db-create:
	@echo "$(COLOR_GREEN)Creating database tables...$(COLOR_RESET)"
ifeq ($(OS),Windows_NT)
	@type database\schema.sql | docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db
else
	@docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db < database/schema.sql
endif
	@echo "$(COLOR_GREEN)✓ Database tables created$(COLOR_RESET)"

## seed: Seed database with initial data
.PHONY: seed
seed:
	@echo "$(COLOR_GREEN)Seeding database...$(COLOR_RESET)"
	@docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db -c "\
		INSERT INTO currencies (id, code, name, symbol, decimal_places, exchange_rate_to_usd, is_active, rate_updated_at, created_at, updated_at) VALUES \
		(gen_random_uuid(), 'USD', 'US Dollar', '\$$', 2, 1.0, true, NOW(), NOW(), NOW()), \
		(gen_random_uuid(), 'EUR', 'Euro', '€', 2, 0.92, true, NOW(), NOW(), NOW()) \
		ON CONFLICT DO NOTHING;"
	@echo "$(COLOR_GREEN)✓ Database seeded$(COLOR_RESET)"

## db-shell: Open PostgreSQL shell
.PHONY: db-shell
db-shell:
	@docker compose exec postgres psql -U ecommerce_user -d ecommerce_db

## redis-shell: Open Redis CLI
.PHONY: redis-shell
redis-shell:
	@docker compose exec redis redis-cli -a redis_password


## ─── Docker (Full Stack) ────────────────────────────────────────────────────
## docker-up: Start API and all services in Docker
.PHONY: docker-up
docker-up:
	@echo "$(COLOR_GREEN)Starting full stack in Docker...$(COLOR_RESET)"
	@docker compose up -d --build

## docker-down: Stop all Docker services and remove volumes
.PHONY: docker-down
docker-down:
	@echo "$(COLOR_YELLOW)Stopping full stack and removing volumes...$(COLOR_RESET)"
	@docker compose down -v

## docker-logs: View logs from all containers
.PHONY: docker-logs
docker-logs:
	@docker compose logs -f
