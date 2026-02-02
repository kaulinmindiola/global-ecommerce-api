# Variables
DOCKER_COMPOSE = docker-compose
GO_CMD = go
BINARY_NAME = ecommerce-api
MAIN_PATH = ./cmd/api

# Colors for output
CYAN = \033[0;36m
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo '$(CYAN)Available commands:$(NC)'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# ==================================================================================== #
# DOCKER COMMANDS
# ==================================================================================== #

.PHONY: docker-up
docker-up: ## Start all Docker containers
	@echo "$(CYAN)Starting Docker containers...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)✓ Containers started successfully$(NC)"
	@echo "$(YELLOW)PostgreSQL:$(NC) localhost:5432"
	@echo "$(YELLOW)Redis:$(NC)      localhost:6379"

.PHONY: docker-up-tools
docker-up-tools: ## Start containers with pgAdmin
	@echo "$(CYAN)Starting Docker containers with tools...$(NC)"
	$(DOCKER_COMPOSE) --profile tools up -d
	@echo "$(GREEN)✓ Containers started successfully$(NC)"
	@echo "$(YELLOW)PostgreSQL:$(NC) localhost:5432"
	@echo "$(YELLOW)Redis:$(NC)      localhost:6379"
	@echo "$(YELLOW)pgAdmin:$(NC)    http://localhost:5050"

.PHONY: docker-down
docker-down: ## Stop all Docker containers
	@echo "$(CYAN)Stopping Docker containers...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)✓ Containers stopped$(NC)"

.PHONY: docker-logs
docker-logs: ## Show Docker logs
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-clean
docker-clean: ## Remove all containers, volumes, and networks
	@echo "$(YELLOW)⚠ This will delete all data!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		$(DOCKER_COMPOSE) down -v --remove-orphans; \
		echo "$(GREEN)✓ Cleanup complete$(NC)"; \
	fi

.PHONY: docker-restart
docker-restart: docker-down docker-up ## Restart all containers

.PHONY: docker-ps
docker-ps: ## Show running containers
	$(DOCKER_COMPOSE) ps

# ==================================================================================== #
# DATABASE COMMANDS
# ==================================================================================== #

.PHONY: db-connect
db-connect: ## Connect to PostgreSQL via psql
	@echo "$(CYAN)Connecting to PostgreSQL...$(NC)"
	docker exec -it ecommerce-postgres psql -U ecommerce_user -d ecommerce_db

.PHONY: db-migrate-up
db-migrate-up: ## Run database migrations (placeholder for now)
	@echo "$(YELLOW)Migration tools will be configured in Phase 2$(NC)"

.PHONY: db-seed
db-seed: ## Seed database with test data (placeholder)
	@echo "$(YELLOW)Seeding tools will be configured in Phase 3$(NC)"

# ==================================================================================== #
# GO COMMANDS
# ==================================================================================== #

.PHONY: run
run: ## Run the application
	$(GO_CMD) run $(MAIN_PATH)/main.go

.PHONY: build
build: ## Build the application
	@echo "$(CYAN)Building application...$(NC)"
	$(GO_CMD) build -o bin/$(BINARY_NAME) $(MAIN_PATH)/main.go
	@echo "$(GREEN)✓ Build complete: bin/$(BINARY_NAME)$(NC)"

.PHONY: test
test: ## Run all tests
	$(GO_CMD) test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test ## Run tests and show coverage
	$(GO_CMD) tool cover -html=coverage.out

.PHONY: lint
lint: ## Run linter
	golangci-lint run --fix

.PHONY: fmt
fmt: ## Format Go code
	$(GO_CMD) fmt ./...
	goimports -w .

# ==================================================================================== #
# DEVELOPMENT COMMANDS
# ==================================================================================== #

.PHONY: dev
dev: docker-up ## Start development environment
	@echo "$(GREEN)✓ Development environment ready!$(NC)"
	@echo "$(YELLOW)Run 'make run' to start the API$(NC)"

.PHONY: setup
setup: ## Initial project setup
	@echo "$(CYAN)Setting up project...$(NC)"
	cp -n .env.example .env || true
	$(GO_CMD) mod download
	$(GO_CMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)✓ Setup complete!$(NC)"
	@echo "$(YELLOW)Run 'make dev' to start development$(NC)"

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out
	$(GO_CMD) clean

# ==================================================================================== #
# DEFAULT TARGET
# ==================================================================================== #

.DEFAULT_GOAL := help
