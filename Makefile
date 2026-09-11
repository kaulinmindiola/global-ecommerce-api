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
MIGRATE_PATH  := ./database/migrations

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
GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./.go/*")
COVERAGE_FILE := coverage.out
COVERAGE_HTML := coverage.html

# Load .env for database URL in Make targets
-include .env
export

# Build the DATABASE_URL from individual vars (used by golang-migrate and psql)
DATABASE_URL ?= postgres://$(DATABASE_USER):$(DATABASE_PASSWORD)@$(DATABASE_HOST):$(DATABASE_PORT)/$(DATABASE_NAME)?sslmode=$(DATABASE_SSL_MODE)

.DEFAULT_GOAL := help

## ─── Help ───────────────────────────────────────────────────────────────────
## help: Display this help message
.PHONY: help
help:
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Global E-commerce API - Available Commands$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"}; \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  $(COLOR_CYAN)%-25s$(COLOR_RESET) %s\n", $$1, $$2 } \
		/^## ───/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 4) }' $(MAKEFILE_LIST)
	@echo ""


## ─── Setup & Development ────────────────────────────────────────────────────
## setup: Initial project setup (creates .env, installs tools)
.PHONY: setup
setup:
	@echo "$(COLOR_GREEN)Setting up project...$(COLOR_RESET)"
	@[ -f .env ] || (cp .env.example .env && echo "✓ .env created from .env.example")
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@mkdir -p bin logs
	@echo "$(COLOR_GREEN)✓ Setup complete!$(COLOR_RESET)"
	@echo "\nNext steps:"
	@echo "  1. Edit .env (set passwords and secrets)"
	@echo "  2. make dev          (start PostgreSQL + Redis)"
	@echo "  3. make migrate-up   (create tables)"
	@echo "  4. make seed         (load sample data)"

## dev: Start PostgreSQL, Redis and Adminer via Docker Compose
.PHONY: dev
dev:
	@echo "$(COLOR_GREEN)Starting infrastructure...$(COLOR_RESET)"
	@docker compose up -d postgres redis adminer
	@echo "$(COLOR_YELLOW)Adminer UI: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Waiting for PostgreSQL to be ready...$(COLOR_RESET)"
	@until docker compose exec postgres pg_isready -U $(DATABASE_USER) -d $(DATABASE_NAME) > /dev/null 2>&1; do \
		printf "."; sleep 1; \
	done
	@echo "\n$(COLOR_GREEN)✓ PostgreSQL ready on localhost:$(DATABASE_PORT)$(COLOR_RESET)"
	@echo "$(COLOR_GREEN)✓ Redis ready on localhost:6379$(COLOR_RESET)"

## stop: Stop all Docker Compose services
.PHONY: stop
stop:
	@echo "$(COLOR_YELLOW)Stopping infrastructure...$(COLOR_RESET)"
	@docker compose down
	@echo "$(COLOR_GREEN)✓ All services stopped$(COLOR_RESET)"

## dev-logs: Follow logs from all Docker Compose services
.PHONY: dev-logs
dev-logs:
	@docker compose logs -f

## run: Run the API server locally (requires .env and running DB)
.PHONY: run
run:
	@echo "$(COLOR_GREEN)Running API server...$(COLOR_RESET)"
	@go run $(LDFLAGS) $(MAIN_PACKAGE)

## build: Compile the production binary for Linux AMD64
.PHONY: build
build:
	@echo "$(COLOR_GREEN)Building production binary...$(COLOR_RESET)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "$(COLOR_GREEN)✓ Binary created: $(BINARY_PATH)$(COLOR_RESET)"
	@ls -lh $(BINARY_PATH)


## ─── Code Quality & Documentation ───────────────────────────────────────────
## format: Format all Go source files
.PHONY: format
format:
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w $(GO_FILES)
	@goimports -w $(GO_FILES)

## lint: Run golangci-lint
.PHONY: lint
lint:
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@golangci-lint run --timeout=5m ./...

## tidy: Tidy and verify go.mod
.PHONY: tidy
tidy:
	@go mod tidy
	@go mod verify

## check: Run all quality checks (format + vet + lint)
.PHONY: check
check: format lint
	@echo "$(COLOR_GREEN)Running go vet...$(COLOR_RESET)"
	@go vet ./...
	@echo "$(COLOR_GREEN)✓ All checks passed$(COLOR_RESET)"

## swagger: Generate Swagger API documentation
.PHONY: swagger
swagger:
	@echo "$(COLOR_GREEN)Generating Swagger docs...$(COLOR_RESET)"
	@swag init -g cmd/api/main.go -o api/docs
	@echo "$(COLOR_GREEN)✓ Swagger docs generated$(COLOR_RESET)"


## ─── Testing ────────────────────────────────────────────────────────────────
## test: Run unit tests with race detector
.PHONY: test
test:
	@echo "$(COLOR_GREEN)Running unit tests...$(COLOR_RESET)"
	@go test -race -count=1 -timeout 30s ./internal/... ./pkg/...

## test-coverage: Run tests and generate HTML coverage report
.PHONY: test-coverage
test-coverage:
	@echo "$(COLOR_GREEN)Generating coverage report...$(COLOR_RESET)"
	@go test -race -count=1 -timeout 30s -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./internal/... ./pkg/...
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(COLOR_GREEN)✓ Coverage report generated: $(COVERAGE_HTML)$(COLOR_RESET)"

## test-integration: Run integration tests (requires running DB + Redis)
.PHONY: test-integration
test-integration:
	@echo "$(COLOR_GREEN)Running integration tests...$(COLOR_RESET)"
	@go test -race -count=1 -timeout 60s -tags=integration ./tests/integration/...

## test-e2e: Run end-to-end tests against the running server
.PHONY: test-e2e
test-e2e:
	@echo "$(COLOR_GREEN)Running E2E tests...$(COLOR_RESET)"
	@go test -race -count=1 -timeout 120s -tags=e2e ./tests/e2e/...

## test-all: Run unit + integration tests
.PHONY: test-all
test-all: test test-integration


## ─── Database Migrations & Seeding ──────────────────────────────────────────
## migrate-up: Run all pending database migrations
.PHONY: migrate-up
migrate-up:
	@echo "$(COLOR_GREEN)Running database migrations...$(COLOR_RESET)"
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" up
	@echo "$(COLOR_GREEN)✓ Migrations applied$(COLOR_RESET)"

## migrate-down: Roll back the last database migration
.PHONY: migrate-down
migrate-down:
	@echo "$(COLOR_YELLOW)Rolling back last migration...$(COLOR_RESET)"
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down 1

## migrate-down-all: Roll back ALL database migrations (destructive!)
.PHONY: migrate-down-all
migrate-down-all:
	@echo "$(COLOR_YELLOW)⚠ This will DROP ALL TABLES. Press Ctrl+C to abort...$(COLOR_RESET)"
	@sleep 3
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" down
	@echo "$(COLOR_GREEN)✓ All migrations rolled back$(COLOR_RESET)"

## migrate-status: Show current migration version
.PHONY: migrate-status
migrate-status:
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" version

## migrate-force: Force set migration version (Usage: make migrate-force VERSION_NUM=4)
.PHONY: migrate-force
migrate-force:
	@[ -n "$(VERSION_NUM)" ] || (echo "Usage: make migrate-force VERSION_NUM=4"; exit 1)
	@migrate -path $(MIGRATE_PATH) -database "$(DATABASE_URL)" force $(VERSION_NUM)
	@echo "$(COLOR_GREEN)✓ Forced migration version to $(VERSION_NUM)$(COLOR_RESET)"

## migrate-create: Create a new migration file pair (Usage: make migrate-create NAME=add_indexes)
.PHONY: migrate-create
migrate-create:
	@[ -n "$(NAME)" ] || (echo "Usage: make migrate-create NAME=migration_name"; exit 1)
	@migrate create -ext sql -dir $(MIGRATE_PATH) -seq $(NAME)
	@echo "$(COLOR_GREEN)✓ Created migration: $(NAME)$(COLOR_RESET)"

## seed: Load all seed data (currencies, users, products)
.PHONY: seed
seed:
	@echo "$(COLOR_GREEN)Seeding database...$(COLOR_RESET)"
	@psql "$(DATABASE_URL)" -f ./database/seeds/001_currencies.sql
	@psql "$(DATABASE_URL)" -f ./database/seeds/002_test_users.sql
	@psql "$(DATABASE_URL)" -f ./database/seeds/003_sample_products.sql
	@echo "$(COLOR_GREEN)✓ Seed data loaded$(COLOR_RESET)"

## db-reset: Full DB reset (drop tables, migrate, seed)
.PHONY: db-reset
db-reset: migrate-down-all migrate-up seed
	@echo "$(COLOR_GREEN)✓ Database reset complete$(COLOR_RESET)"


## ─── Shell Access ───────────────────────────────────────────────────────────
## db-shell: Open an interactive PostgreSQL shell
.PHONY: db-shell
db-shell:
	@docker compose exec postgres psql -U $(DATABASE_USER) -d $(DATABASE_NAME)

## redis-shell: Open an interactive Redis CLI
.PHONY: redis-shell
redis-shell:
	@docker compose exec redis redis-cli -a $(REDIS_PASSWORD)


## ─── Docker (Full Stack) ────────────────────────────────────────────────────
## docker-build: Build the production Docker image
.PHONY: docker-build
docker-build:
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.
	@echo "$(COLOR_GREEN)✓ Image: $(BINARY_NAME):$(VERSION)$(COLOR_RESET)"

## docker-up: Run the full stack in Docker (API + PostgreSQL + Redis)
.PHONY: docker-up
docker-up:
	@echo "$(COLOR_GREEN)Starting full stack in Docker...$(COLOR_RESET)"
	@docker compose up -d --build

## docker-down: Remove all project Docker images and volumes
.PHONY: docker-down
docker-down:
	@echo "$(COLOR_YELLOW)Cleaning Docker resources...$(COLOR_RESET)"
	@docker compose down -v --rmi local
	@echo "$(COLOR_GREEN)✓ Docker resources cleaned$(COLOR_RESET)"

# ==================================================================================== #
# DOCKER PRODUCTION COMMANDS
# ==================================================================================== #

PROD_COMPOSE = docker compose -f docker-compose.prod.yaml
TEST_COMPOSE = docker compose -f docker-compose.test.yaml
IMAGE_NAME   := global-ecommerce-api
IMAGE_REPO   := ghcr.io/kaulinmindiola/$(IMAGE_NAME)
IMAGE_TAG    := $(shell git rev-parse --short HEAD)

.PHONY: docker-build-prod
docker-build-prod: ## Build production Docker image
	@echo '$(COLOR_CYAN)Building production image...$(COLOR_RESET)'
	@chmod +x scripts/docker/build.sh
	@./scripts/docker/build.sh
	@echo '$(COLOR_GREEN)✓ Production image built$(COLOR_RESET)'

.PHONY: docker-build-push
docker-build-push: ## Build and push production image to registry
	@echo '$(COLOR_CYAN)Building and pushing production image...$(COLOR_RESET)'
	@./scripts/docker/build.sh --push
	@echo '$(COLOR_GREEN)✓ Image pushed to registry$(COLOR_RESET)'

.PHONY: docker-build-scan
docker-build-scan: ## Build production image and run security scan
	@echo '$(COLOR_CYAN)Building and scanning production image...$(COLOR_RESET)'
	@./scripts/docker/build.sh --scan
	@echo '$(COLOR_GREEN)✓ Build and scan complete$(COLOR_RESET)'

.PHONY: docker-prod-up
docker-prod-up: ## Start production stack with auto-injected tag
	@echo '$(COLOR_CYAN)Starting production stack...$(COLOR_RESET)'
	IMAGE_TAG=$(IMAGE_TAG) $(PROD_COMPOSE) up -d
	@echo '$(COLOR_GREEN)✓ Production stack started$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)  • API:    http://localhost:80$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)  • HTTPS:  https://localhost:443$(COLOR_RESET)'

.PHONY: docker-prod-down
docker-prod-down: ## Stop production stack
	@echo '$(COLOR_CYAN)Stopping production stack...$(COLOR_RESET)'
	@$(PROD_COMPOSE) down
	@echo '$(COLOR_GREEN)✓ Production stack stopped$(COLOR_RESET)'

.PHONY: docker-prod-logs
docker-prod-logs: ## Show production logs
	@$(PROD_COMPOSE) logs -f

.PHONY: docker-prod-ps
docker-prod-ps: ## Show production container status
	@$(PROD_COMPOSE) ps

.PHONY: docker-test
docker-test: ## Run tests in Docker (isolated environment)
	@echo '$(COLOR_CYAN)Running tests in Docker...$(COLOR_RESET)'
	@$(TEST_COMPOSE) up --build --abort-on-container-exit test-runner
	@$(TEST_COMPOSE) down -v
	@echo '$(COLOR_GREEN)✓ Docker tests complete$(COLOR_RESET)'

.PHONY: docker-image-size
docker-image-size: ## Check production image size
	@echo '$(COLOR_CYAN)Production image size:$(COLOR_RESET)'
	@docker image ls $(IMAGE_REPO) --format "table {{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"

.PHONY: docker-image-inspect
docker-image-inspect: ## Inspect production image layers
	@echo '$(COLOR_CYAN)Image layer analysis:$(COLOR_RESET)'
	@docker history $(IMAGE_REPO):$(IMAGE_TAG) --human --format \
    "table {{.CreatedBy}}\t{{.Size}}"

.PHONY: docker-trivy-scan
docker-trivy-scan: ## Run Trivy security scan on production image
	@echo '$(COLOR_CYAN)Running Trivy security scan...$(COLOR_RESET)'
	@docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		aquasec/trivy:latest image \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		$(IMAGE_REPO):$(IMAGE_TAG)
	@echo '$(COLOR_GREEN)✓ Security scan complete$(COLOR_RESET)'

.PHONY: docker-dev
docker-dev: ## Start development with hot reload (Air)
	@echo '$(COLOR_CYAN)Starting development with hot reload...$(COLOR_RESET)'
	@docker compose -f docker-compose.yaml up --build api-dev


## ─── Cleanup ────────────────────────────────────────────────────────────────
## clean: Remove build artifacts and test cache
.PHONY: clean
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf bin/ $(COVERAGE_FILE) $(COVERAGE_HTML) api/docs/
	@go clean -testcache -cache -modcache
	@echo "$(COLOR_GREEN)✓ Cleaned$(COLOR_RESET)"

# ==================================================================================== #
# DOCUMENTATION COMMANDS
# ==================================================================================== #

.PHONY: swagger-init
swagger-init:
	@echo '$(COLOR_CYAN)Initializing Swagger documentation...$(COLOR_RESET)'
	@swag init \
		-g cmd/api/main.go \
		-o api/docs \
		--parseDependency \
		--parseInternal
	@echo '$(COLOR_GREEN)✓ Swagger docs generated in api/docs/$(COLOR_RESET)'

.PHONY: swagger-validate
swagger-validate: ## Validate OpenAPI spec
	@echo '$(COLOR_CYAN)Validating OpenAPI specification...$(COLOR_RESET)'
	@docker run --rm \
		-v $(PWD)/api/docs:/workspace \
		openapitools/openapi-generator-cli:latest \
		validate -i /workspace/swagger.yaml
	@echo '$(COLOR_GREEN)✓ OpenAPI spec is valid$(COLOR_RESET)'

.PHONY: swagger-serve
swagger-serve: swagger-init ## Generate docs and start server with Swagger UI
	@echo '$(COLOR_CYAN)Starting server with Swagger UI...$(COLOR_RESET)'
	@echo '$(COLOR_YELLOW)Swagger UI available at: $(COLOR_GREEN)http://localhost:8080/swagger$(COLOR_RESET)'
	@$(MAKE) run

.PHONY: docs
docs: swagger-init ## Alias for swagger-init

# ==================================================================================== #
# CI/CD COMMANDS
# ==================================================================================== #

.PHONY: ci-local
ci-local: format vet lint test build ## Run full CI pipeline locally
	@echo '$(COLOR_GREEN)✓ Local CI pipeline passed!$(COLOR_RESET)'

.PHONY: pre-commit
pre-commit: format vet lint test ## Run pre-commit checks (fast)
	@echo '$(COLOR_GREEN)✓ Pre-commit checks passed!$(COLOR_RESET)'

.PHONY: check-secrets
check-secrets: ## Check for accidentally committed secrets
	@echo '$(COLOR_CYAN)Scanning for secrets...$(COLOR_RESET)'
	@docker run --rm -v $(PWD):/repo \
		trufflesecurity/trufflehog:latest \
		filesystem /repo --only-verified 2>&1 || true
	@echo '$(COLOR_GREEN)✓ Secret scan complete$(COLOR_RESET)'

.PHONY: security-scan
security-scan: ## Run gosec security scanner
	@echo '$(COLOR_CYAN)Running security scan...$(COLOR_RESET)'
	@gosec -severity medium -confidence medium ./... 2>&1 || true
	@echo '$(COLOR_GREEN)✓ Security scan complete$(COLOR_RESET)'

.PHONY: coverage-report
coverage-report: ## Generate and open coverage report
	@echo '$(COLOR_CYAN)Generating coverage report...$(COLOR_RESET)'
	@go test -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo '$(COLOR_GREEN)✓ Coverage report: coverage.html$(COLOR_RESET)'
