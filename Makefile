# ============================================================================
# GLOBAL E-COMMERCE API - MAKEFILE
# ============================================================================
# Professional build automation for development workflow
#
# Usage:
#   make help          Show this help message
#   make setup         Initial project setup
#   make dev           Start development environment
#   make test          Run all tests
#   make build         Build production binary
# ============================================================================

.PHONY: help setup dev stop test build clean lint format migrate-up migrate-down docker-build docker-up docker-down

# Default target
.DEFAULT_GOAL := help

# Variables
APP_NAME := global-ecommerce-api
BINARY_NAME := ecommerce-api
DOCKER_IMAGE := $(APP_NAME):latest
GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)$(COLOR_BLUE)Global E-commerce API - Available Commands$(COLOR_RESET)"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
	@echo ""

## setup: Initial project setup (run once)
setup:
	@echo "$(COLOR_GREEN)Setting up project...$(COLOR_RESET)"
	@echo "Creating .env file from template..."
	@cp -n .env.example .env || true
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Creating required directories..."
	@mkdir -p bin logs
	@echo "$(COLOR_GREEN)✓ Setup complete!$(COLOR_RESET)"

## dev: Start development environment (Docker + hot-reload)
dev:
	@echo "$(COLOR_GREEN)Starting development environment...$(COLOR_RESET)"
	@docker-compose up -d
	@echo "$(COLOR_GREEN)✓ Services started!$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Database UI: http://localhost:8080$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)PostgreSQL: localhost:5432$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Redis: localhost:6379$(COLOR_RESET)"

## stop: Stop development environment
stop:
	@echo "$(COLOR_YELLOW)Stopping development environment...$(COLOR_RESET)"
	@docker-compose down
	@echo "$(COLOR_GREEN)✓ Services stopped$(COLOR_RESET)"

## build: Build production binary
build:
	@echo "$(COLOR_GREEN)Building production binary...$(COLOR_RESET)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags="-w -s -X main.Version=$(shell git describe --tags --always --dirty)" \
		-o bin/$(BINARY_NAME) \
		./cmd/api
	@echo "$(COLOR_GREEN)✓ Binary created: bin/$(BINARY_NAME)$(COLOR_RESET)"

## run: Run API locally (without Docker)
run:
	@echo "$(COLOR_GREEN)Running API server...$(COLOR_RESET)"
	@go run cmd/api/main.go

## test: Run all tests with coverage
test:
	@echo "$(COLOR_GREEN)Running tests...$(COLOR_RESET)"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@echo "$(COLOR_GREEN)✓ Tests complete$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Coverage report: coverage.txt$(COLOR_RESET)"

## test-coverage: Run tests and display coverage in browser
test-coverage: test
	@go tool cover -html=coverage.txt

## lint: Run linter (golangci-lint)
lint:
	@echo "$(COLOR_GREEN)Running linter...$(COLOR_RESET)"
	@golangci-lint run --timeout=5m ./...
	@echo "$(COLOR_GREEN)✓ Linting complete$(COLOR_RESET)"

## format: Format Go code
format:
	@echo "$(COLOR_GREEN)Formatting code...$(COLOR_RESET)"
	@gofmt -s -w $(GO_FILES)
	@goimports -w $(GO_FILES)
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

## migrate-up: Run database migrations (up)
migrate-up:
	@echo "$(COLOR_GREEN)Running database migrations...$(COLOR_RESET)"
	@docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db < database/schema.sql
	@echo "$(COLOR_GREEN)✓ Migrations applied$(COLOR_RESET)"

## db-create: Create database tables from schema
db-create:
	@echo "$(COLOR_GREEN)Creating database tables...$(COLOR_RESET)"
ifeq ($(OS),Windows_NT)
	@type database\schema.sql | docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db
else
	@docker exec -i ecommerce_postgres psql -U ecommerce_user -d ecommerce_db < database/schema.sql
endif
	@echo "$(COLOR_GREEN)✓ Database tables created$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Verifying tables...$(COLOR_RESET)"
	@docker exec ecommerce_postgres psql -U ecommerce_user -d ecommerce_db -c "\dt"

## migrate-down: Rollback database migrations (down)
migrate-down:
	@echo "$(COLOR_YELLOW)Rolling back migrations...$(COLOR_RESET)"
	@echo "Manual rollback required - check database/migrations/*.down.sql"

## seed: Seed database with test data
seed:
	@echo "$(COLOR_GREEN)Seeding database...$(COLOR_RESET)"
	@docker-compose exec postgres psql -U ecommerce_user -d ecommerce_db -c "\
		INSERT INTO currencies (code, name, symbol, decimal_places) VALUES \
		('USD', 'US Dollar', '\$$', 2), \
		('EUR', 'Euro', '€', 2), \
		('COP', 'Colombian Peso', '\$$', 2) \
		ON CONFLICT DO NOTHING;"
	@echo "$(COLOR_GREEN)✓ Database seeded$(COLOR_RESET)"

## clean: Remove build artifacts and caches
clean:
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf bin/
	@rm -rf dist/
	@rm -f coverage.txt
	@go clean -cache -testcache -modcache
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

## docker-build: Build Docker image
docker-build:
	@echo "$(COLOR_GREEN)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(DOCKER_IMAGE) -f deployments/docker/Dockerfile .
	@echo "$(COLOR_GREEN)✓ Image built: $(DOCKER_IMAGE)$(COLOR_RESET)"

## docker-up: Start all services with Docker Compose
docker-up:
	@docker-compose up -d --build

## docker-down: Stop and remove all Docker containers
docker-down:
	@docker-compose down -v

## docker-logs: View logs from all containers
docker-logs:
	@docker-compose logs -f

## deps: Download and tidy Go dependencies
deps:
	@echo "$(COLOR_GREEN)Updating dependencies...$(COLOR_RESET)"
	@go mod download
	@go mod tidy
	@go mod verify
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

## swagger: Generate Swagger documentation
swagger:
	@echo "$(COLOR_GREEN)Generating Swagger docs...$(COLOR_RESET)"
	@swag init -g cmd/api/main.go -o api/docs
	@echo "$(COLOR_GREEN)✓ Swagger docs generated$(COLOR_RESET)"

## db-shell: Open PostgreSQL shell
db-shell:
	@docker-compose exec postgres psql -U ecommerce_user -d ecommerce_db

## redis-shell: Open Redis CLI
redis-shell:
	@docker-compose exec redis redis-cli -a redis_password
