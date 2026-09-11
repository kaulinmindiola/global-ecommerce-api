# 🌍 Global E-commerce Microservices API

> A production-grade RESTful API for international e-commerce platforms with multi-currency support, timezone-aware operations, and enterprise-level architecture.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=flat&logo=redis)](https://redis.io)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 📖 Table of Contents

- [Overview](#-overview)
- [Key Features](#-key-features)
- [Architecture](#-architecture)
- [Technology Stack](#-technology-stack)
- [Getting Started](#-getting-started)
- [Project Structure](#-project-structure)
- [API Documentation](#-api-documentation)
- [Database Schema](#-database-schema)
- [Development](#-development)
- [Testing](#-testing)
- [Deployment](#-deployment)
- [Contributing](#-contributing)
- [License](#-license)
- [Author](#-author)

---

## 🎯 Overview

This API demonstrates **senior-level backend engineering** for global commerce platforms. It solves complex real-world challenges faced by international businesses:

- **Multi-currency pricing** with historical exchange rate tracking for financial compliance
- **Timezone-aware operations** ensuring consistent timestamps across global customer bases
- **Scalable architecture** with Redis caching and proper separation of concerns
- **Production-ready code** following Clean Architecture principles

### 🔑 Key Features

- ✅ **RESTful API** with OpenAPI/Swagger documentation
- ✅ **Multi-currency support** (USD, EUR, COP, GBP, JPY) with real-time conversion
- ✅ **UTC-based timestamps** for global consistency
- ✅ **Redis caching layer** for performance optimization
- ✅ **Clean Architecture** with dependency inversion
- ✅ **Comprehensive error handling** with detailed error codes
- ✅ **JWT authentication** for secure API access
- ✅ **Docker containerization** for easy deployment
- ✅ **Professional documentation** with architecture diagrams

---

## 🏗️ Architecture

This project follows **Clean Architecture** principles with clear separation between business logic, data access, and infrastructure concerns.

```mermaid
graph TB
    subgraph "Client Layer"
        A1[Web App]
        A2[Mobile App]
        A3[Third Party API]
    end

    subgraph "API Layer"
        B1[Load Balancer]
        B2[API Server - Go]
    end

    subgraph "Application Layer"
        C1[HTTP Router]
        C2[Middleware]
        C3[Handlers]
        C4[Services]
    end

    subgraph "Data Layer"
        D1[Repository Layer]
        D2[(PostgreSQL)]
        D3[(Redis Cache)]
    end

    A1 & A2 & A3 --> B1
    B1 --> B2
    B2 --> C1
    C1 --> C2
    C2 --> C3
    C3 --> C4
    C4 --> D1
    D1 --> D2
    D1 --> D3

    style B2 fill:#00ADD8
    style D2 fill:#336791
    style D3 fill:#DC382D
```

### Architecture Layers

1. **Domain Layer** (`internal/domain/`) - Core business entities
2. **Application Layer** (`internal/service/`) - Business logic and use cases
3. **Interface Adapters** (`internal/repository/`) - Data access abstractions
4. **Infrastructure** (`internal/handler/`, `cmd/`) - External interfaces

📚 **Detailed Documentation:**
- [Complete Architecture Guide](docs/ARCHITECTURE.md)
- [System Diagrams](docs/diagrams.md)

---

## 💻 Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Language** | Go | 1.22+ | High-performance backend |
| **Database** | PostgreSQL | 16 | Relational data storage with ACID compliance |
| **Cache** | Redis | 7 | Session management and data caching |
| **API Framework** | Chi Router | 5.x | Lightweight HTTP routing and middleware |
| **Authentication** | JWT | - | Stateless token-based authentication |
| **Validation** | go-playground/validator | 10.x | Request input validation |
| **Documentation** | OpenAPI 3.0 | - | API specification and Swagger UI |

### Development Tools

| Tool | Purpose |
|------|---------|
| Docker & Docker Compose | Local development environment |
| golangci-lint | Code quality and linting |
| testify | Testing framework with assertions |
| Makefile | Build automation and task runner |
| Adminer | Database management UI |

---

## 🚀 Getting Started

### Prerequisites

Ensure you have the following installed:

- [Go 1.22+](https://golang.org/dl/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop)
- [Git](https://git-scm.com/downloads)
- [Make](https://www.gnu.org/software/make/) (usually pre-installed on Linux/Mac)

### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/kaulinmindiola/global-ecommerce-api.git
   cd global-ecommerce-api
   ```

2. **Run initial setup**
   ```bash
   make setup
   ```
   This will:
   - Create `.env` from `.env.example`
   - Download Go dependencies
   - Install development tools

3. **Start the development environment**
   ```bash
   make dev
   ```
   This starts:
   - PostgreSQL (port 5432)
   - Redis (port 6379)
   - Adminer database UI (port 8080)

4. **Verify services are running**
   ```bash
   docker ps
   ```
   You should see 3 containers running:
   - `ecommerce_postgres`
   - `ecommerce_redis`
   - `ecommerce_adminer`

5. **Access services**
   - **Database UI (Adminer)**: http://localhost:8080
     - System: PostgreSQL
     - Server: postgres
     - Username: ecommerce_user
     - Password: ecommerce_password
     - Database: ecommerce_db
   
   - **API Server** (when implemented): http://localhost:8000
   - **API Documentation** (when implemented): http://localhost:8000/swagger

### Available Commands

```bash
make help          # Show all available commands
make setup         # Initial project setup
make dev           # Start development environment
make stop          # Stop all services
make test          # Run tests with coverage
make lint          # Run code linter
make format        # Format Go code
make build         # Build production binary
make clean         # Remove build artifacts
make db-shell      # Open PostgreSQL shell
make redis-shell   # Open Redis CLI
```

---

## 📁 Project Structure

```
global-ecommerce-api/
│
├── cmd/                          # Application entry points
│   └── api/
│       └── main.go              # Main API server entry point
│
├── internal/                     # Private application code
│   ├── domain/                  # Business entities (Product, Order, User)
│   │   ├── user.go
│   │   ├── product.go
│   │   ├── order.go
│   │   └── currency.go
│   │
│   ├── repository/              # Data access layer
│   │   ├── interfaces.go        # Repository contracts
│   │   ├── postgres/            # PostgreSQL implementations
│   │   │   ├── user_repository.go
│   │   │   ├── product_repository.go
│   │   │   ├── order_repository.go
│   │   │   └── currency_repository.go
│   │   └── redis/               # Redis cache implementations
│   │       └── cache_repository.go
│   │
│   ├── service/                 # Business logic layer
│   │   ├── user_service.go
│   │   ├── product_service.go
│   │   ├── order_service.go
│   │   └── currency_service.go
│   │
│   ├── handler/                 # HTTP handlers (controllers)
│   │   ├── user_handler.go
│   │   ├── product_handler.go
│   │   └── order_handler.go
│   │
│   └── middleware/              # HTTP middleware
│       ├── auth.go              # JWT authentication
│       ├── logging.go           # Request logging
│       └── cors.go              # CORS configuration
│
├── pkg/                         # Public libraries (reusable)
│   ├── logger/                  # Logging utilities
│   ├── validator/               # Input validation
│   └── errors/                  # Custom error types
│
├── config/                      # Configuration management
│   ├── config.go               # Configuration loader
│   └── config.yaml             # Default settings
│
├── database/                    # Database related files
│   ├── schema.sql              # Complete database schema
│   ├── migrations/             # SQL migration files
│   └── seeds/                  # Seed data for testing
│
├── api/                        # API specifications
│   ├── openapi/                # OpenAPI 3.0 specs
│   │   └── openapi.yaml
│   └── docs/                   # Generated Swagger docs
│
├── docs/                       # Project documentation
│   ├── ARCHITECTURE.md         # Architecture guide
│   └── diagrams.md             # System diagrams
│
├── tests/                      # Test files
│   ├── integration/            # Integration tests
│   └── fixtures/               # Test data
│
├── scripts/                    # Utility scripts
│   ├── setup.sh               # Environment setup
│   ├── migrate.sh             # Migration runner
│   └── seed.sh                # Database seeding
│
├── deployments/                # Deployment configurations
│   └── docker/
│       └── Dockerfile         # Production container image
│
├── .github/                    # GitHub specific files
│   └── workflows/
│       └── ci.yaml            # CI/CD pipeline
│
├── .env.example               # Environment variables template
├── .gitignore                 # Git ignore rules
├── docker-compose.yaml        # Local development setup
├── Makefile                   # Build automation
├── go.mod                     # Go module dependencies
├── go.sum                     # Go dependencies checksum
└── README.md                  # This file

```

**Design Principles:**
- `cmd/`: Application entry points only, minimal code
- `internal/`: Private code that cannot be imported by other projects
- `pkg/`: Reusable libraries that could be extracted
- Clear separation between layers (domain → service → handler)

---

## 📊 API Documentation

### Endpoint Overview

The API provides **24 RESTful endpoints** organized by domain:

| Domain | Endpoints | Description |
|--------|-----------|-------------|
| **Health** | 2 | System health checks and version info |
| **Authentication** | 4 | User registration, login, token refresh |
| **Users** | 4 | User profile management |
| **Currencies** | 4 | Currency listings and conversion |
| **Products** | 6 | Product catalog management |
| **Orders** | 5 | Order creation and management |

### Example Endpoints

```bash
# Health Check
GET /api/v1/health

# User Authentication
POST /api/v1/auth/register
POST /api/v1/auth/login

# Products (Public)
GET /api/v1/products
GET /api/v1/products/{id}

# Orders (Authenticated)
POST /api/v1/orders
GET /api/v1/orders
GET /api/v1/orders/{id}

# Currency Conversion
GET /api/v1/currencies/convert?from=USD&to=EUR&amount=100
```

### Request/Response Examples

**Create Order (Multi-currency)**
```bash
POST /api/v1/orders
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "currency": "EUR",
  "items": [
    {
      "product_id": "p1234567-e89b-12d3-a456-426614174000",
      "quantity": 2
    }
  ]
}
```

**Response:**
```json
{
  "id": "o1234567-e89b-12d3-a456-426614174000",
  "order_number": "ORD-2025-000123",
  "currency": "EUR",
  "exchange_rate": 0.9250,
  "total_amount": 662.47,
  "status": "pending",
  "created_at": "2025-02-22T16:30:00Z"
}
```

📚 **Complete API specification:** See [API Contracts Documentation](api/openapi/openapi.yaml)

---

## 🗄️ Database Schema

### Entity Relationship Diagram

```mermaid
erDiagram
    currencies ||--o{ products : "base_currency"
    currencies ||--o{ users : "preferred_currency"
    currencies ||--o{ orders : "currency"
    users ||--o{ orders : "places"
    products ||--o{ order_items : "contains"
    orders ||--|{ order_items : "has"
    
    currencies {
        uuid id PK
        string code
        string name
        string symbol
    }
    
    users {
        uuid id PK
        string email
        string full_name
        uuid preferred_currency_id FK
        string preferred_timezone
    }
    
    products {
        uuid id PK
        string name
        string sku
        decimal base_price
        uuid base_currency_id FK
        int stock_quantity
    }
    
    orders {
        uuid id PK
        string order_number
        uuid user_id FK
        uuid currency_id FK
        decimal exchange_rate
        decimal total_amount
    }
```

### Key Design Decisions

1. **NUMERIC for prices** - Avoids floating-point precision errors
2. **TIMESTAMPTZ for all dates** - Timezone-aware, stored in UTC
3. **Exchange rate snapshot** - Stored in orders for financial compliance
4. **UUIDs for all IDs** - Better for distributed systems
5. **Referential integrity** - Foreign keys with proper constraints

📄 **Complete schema:** [database/schema.sql](database/schema.sql)

---

## 🔧 Development

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

**Key variables:**
```env
# Application
APP_ENV=development
APP_PORT=8000

# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=ecommerce_user
DATABASE_PASSWORD=ecommerce_password
DATABASE_NAME=ecommerce_db

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password

# JWT
JWT_SECRET=your-super-secret-key
JWT_EXPIRATION=24h
```

### Running Locally

**Option 1: With Docker (Recommended)**
```bash
make dev
# Services run in containers
```

**Option 2: Native (requires local PostgreSQL & Redis)**
```bash
# Start database services
docker-compose up -d postgres redis

# Run API server locally
make run
```

### Database Management

```bash
# Access PostgreSQL CLI
make db-shell

# Access Redis CLI
make redis-shell

# View database in browser
# Open http://localhost:8080
```

### Code Quality

```bash
# Format code
make format

# Run linter
make lint

# Run all checks before commit
make format && make lint && make test
```

---

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run specific package
go test -v ./internal/service/...
```

### Test Coverage Goal

- **Target:** >80% code coverage
- **Focus areas:**
  - Business logic (services)
  - Currency conversion calculations
  - Order total calculations
  - Error handling

### Testing Strategy

1. **Unit Tests** - Service layer and business logic
2. **Integration Tests** - Database operations
3. **API Tests** - HTTP handlers (future)
4. **E2E Tests** - Complete workflows (future)

---

## 🚀 Deployment

### Docker Build

```bash
# Build production image
make docker-build

# Run container
docker run -p 8000:8000 \
  -e DATABASE_URL=postgresql://user:pass@host:5432/db \
  global-ecommerce-api:latest
```

### Cloud Deployment (Future)

**Recommended Platforms:**
- **AWS**: ECS + RDS + ElastiCache
- **Google Cloud**: Cloud Run + Cloud SQL + Memorystore
- **Railway**: Simplified deployment with managed PostgreSQL
- **Render**: Zero-config deployment

**Architecture in Production:**
```
Internet → Load Balancer → API Instances → RDS (PostgreSQL)
                                        → ElastiCache (Redis)
```

---

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. **Fork the repository**F
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Write tests** for new functionality
4. **Follow Go conventions** and run `make lint`
5. **Commit with conventional commits** (`feat:`, `fix:`, `docs:`)
6. **Push to your fork** and create a **Pull Request**

### Coding Standards

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Write clear commit messages
- Add comments for complex logic
- Maintain >80% test coverage

---

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

## 👨‍💻 Author

**Kaulin Mindiola**

- GitHub: [@kaulinmindiola](https://github.com/kaulinmindiola)
- LinkedIn: [Kaulin Mindiola](https://linkedin.com/in/kaulinmindiola)
- Email: kaulinmindiola@gmail.com

---

## 🙏 Acknowledgments

- Built following Clean Architecture principles by Robert C. Martin
- Inspired by production-grade systems from enterprise companies
- Database design influenced by PostgreSQL best practices
- API design following RESTful conventions and OpenAPI standards

---

## 📈 Project Status

**Current Phase:** ✅ Phase 1 - Architecture & Design (Complete)

### Completed
- ✅ Database schema design
- ✅ API contract definition (24 endpoints)
- ✅ Architecture documentation
- ✅ Error handling specification
- ✅ Development environment setup
- ✅ Project structure scaffolding

### In Progress
- 🚧 Phase 2 - Implementation (Next)

### Roadmap
- ⏳ Core business logic implementation
- ⏳ Unit testing (>80% coverage)
- ⏳ Integration testing
- ⏳ OpenAPI/Swagger documentation
- ⏳ CI/CD pipeline (GitHub Actions)
- ⏳ Cloud deployment

---

## 🌟 Why This Project?

This API demonstrates **production-ready engineering practices** suitable for international companies:

1. **Global-First Design** - Multi-currency and timezone support from day one
2. **Financial Accuracy** - NUMERIC types and exchange rate snapshots
3. **Scalable Architecture** - Stateless design with caching strategy
4. **Clean Code** - Follows SOLID principles and Clean Architecture
5. **Professional Documentation** - Architecture diagrams and clear contracts
6. **Enterprise Patterns** - Repository pattern, dependency injection, middleware
7. **Security-Conscious** - JWT auth, input validation, prepared statements

**Perfect for portfolios targeting:** Backend Engineer, Cloud Engineer, Microservices Developer roles in USD/EUR markets.

---

<p align="center">
  <strong>Built with 💙 by Kaulin Mindiola</strong><br>
  <sub>Demonstrating senior-level backend engineering for the global market</sub>
</p>
