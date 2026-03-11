# 🏗️ Architecture Documentation

## Table of Contents
1. [System Overview](#system-overview)
2. [Architecture Layers](#architecture-layers)
3. [Data Flow](#data-flow)
4. [Technology Stack](#technology-stack)
5. [Design Patterns](#design-patterns)
6. [Scalability Strategy](#scalability-strategy)

---

## System Overview

The Global E-commerce API follows **Clean Architecture** principles with clear separation between business logic, data access, and infrastructure concerns.

### High-Level Architecture

```
┌─────────────┐
│   Clients   │
└──────┬──────┘
       │
       ▼
┌──────────────┐     ┌───────────┐
│  API Server  │────▶│   Redis   │
│  (Go/Chi)    │     │  (Cache)  │
└──────┬───────┘     └───────────┘
       │
       ▼
┌──────────────┐
│  PostgreSQL  │
│  (Primary)   │
└──────────────┘
```

---

## Architecture Layers

### 1. Domain Layer (Core)
**Location:** `internal/domain/`

Contains business entities and core business rules:
- `Product`: Catalog items with multi-currency pricing
- `Order`: Transactions with exchange rate snapshots
- `User`: Customer profiles with timezone preferences
- `Currency`: Supported currencies and conversion rates

**Key Principle:** No dependencies on external frameworks or databases.

### 2. Application Layer (Use Cases)
**Location:** `internal/service/`

Implements business logic and use cases:
- `OrderService`: Order creation, calculation, validation
- `CurrencyService`: Exchange rate conversion
- `AuthService`: JWT token generation and validation

**Dependencies:** Only on domain entities and repository interfaces.

### 3. Interface Adapters
**Location:** `internal/repository/`

Defines contracts (ports) and implements adapters:
- `interfaces.go`: Repository interfaces
- `postgres/`: PostgreSQL implementations
- `redis/`: Redis cache implementations

### 4. Infrastructure Layer
**Location:** `internal/handler/`, `cmd/api/`

Handles external communication:
- HTTP handlers (controllers)
- Middleware (auth, logging, CORS)
- Database drivers
- HTTP server setup

---

## Data Flow

### Example: Creating an Order

```
1. Client sends POST /api/v1/orders
2. Auth middleware validates JWT token
3. Handler parses and validates request
4. OrderService orchestrates business logic:
   a. Check Redis cache for product prices
   b. Fetch from PostgreSQL if cache miss
   c. Get current exchange rate
   d. Calculate total with tax and shipping
   e. Validate stock availability
   f. Create order in database transaction
5. Response returned to client
```

### Caching Strategy

| Data Type | Cache Key | TTL | Purpose |
|-----------|-----------|-----|---------|
| Products | `product:{id}` | 1 hour | Reduce DB load |
| Exchange Rates | `rate:{from}:{to}` | 1 hour | Minimize API calls |
| User Sessions | `session:{token}` | 24 hours | JWT validation |
| Product Lists | `products:page:{n}` | 5 minutes | List pagination |

**Cache Invalidation:**
- Product updated → Delete `product:{id}`
- Order created → Delete `products:page:*` (stock changed)
- Exchange rate updated → Delete `rate:*`

---

## Technology Stack

### Core Technologies

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Language | Go | 1.22+ | High-performance backend |
| HTTP Framework | Chi | 5.x | Routing and middleware |
| Database | PostgreSQL | 16 | Relational data storage |
| Cache | Redis | 7 | Session and data caching |
| Authentication | JWT | - | Stateless auth |
| Validation | go-playground/validator | 10.x | Input validation |

### Development Tools

| Tool | Purpose |
|------|---------|
| Docker Compose | Local environment |
| golangci-lint | Code quality |
| testify | Testing framework |
| Swagger | API documentation |
| Makefile | Build automation |

---

## Design Patterns

### 1. Repository Pattern
Abstracts data access logic:

```go
type ProductRepository interface {
    GetByID(ctx context.Context, id string) (*Product, error)
    Create(ctx context.Context, product *Product) error
    Update(ctx context.Context, product *Product) error
}
```

**Benefits:**
- Easy to mock for testing
- Database-agnostic business logic
- Single source of truth for queries

### 2. Dependency Injection
Services receive dependencies through constructors:

```go
func NewOrderService(
    productRepo repository.ProductRepository,
    orderRepo repository.OrderRepository,
    cache cache.Service,
) *OrderService {
    return &OrderService{
        productRepo: productRepo,
        orderRepo:   orderRepo,
        cache:       cache,
    }
}
```

### 3. Middleware Pattern
Cross-cutting concerns handled via middleware:

```go
router.Use(middleware.Logger)
router.Use(middleware.CORS)
router.Use(middleware.RateLimiter)

// Protected routes
router.Group(func(r chi.Router) {
    r.Use(middleware.Authenticate)
    r.Post("/orders", orderHandler.Create)
})
```

---

## Scalability Strategy

### Horizontal Scaling
- **Stateless API**: No server-side sessions (JWT)
- **Load Balancer**: Distribute traffic across instances
- **Connection Pooling**: Reuse database connections

### Vertical Scaling
- **Database**: Read replicas for queries
- **Cache**: Redis cluster mode
- **Compute**: Increase instance size

### Performance Optimizations

1. **Database Indexes**
   - `idx_products_sku` on products(sku)
   - `idx_orders_user` on orders(user_id)
   - `idx_orders_created` on orders(created_at)

2. **Query Optimization**
   - Pagination for large result sets
   - Selective field loading (avoid SELECT *)
   - Batch operations where applicable

3. **Caching Strategy**
   - Cache hot data (popular products)
   - Cache-aside pattern
   - TTL based on data volatility

4. **Connection Management**
   - Connection pooling (max 50 connections)
   - Idle connection timeout (5 minutes)
   - Query timeout (30 seconds)

---

## Security Considerations

### Authentication & Authorization
- JWT tokens with 24-hour expiration
- Password hashing with bcrypt (cost 12)
- Role-based access control (User, Admin)

### Data Protection
- SSL/TLS for database connections (production)
- Secrets stored in environment variables
- No sensitive data in logs

### API Security
- Rate limiting (100 requests/minute)
- CORS configuration
- Input validation on all endpoints
- SQL injection prevention (parameterized queries)

---

## Monitoring & Observability

### Logging
- Structured logging (JSON format)
- Request ID tracking
- Error stack traces

### Metrics (Future)
- Request latency (p50, p95, p99)
- Error rates by endpoint
- Database query performance
- Cache hit rate

### Health Checks
- `GET /api/v1/health`
  - Database connectivity
  - Redis connectivity
  - API version

---

## Future Enhancements

1. **Event-Driven Architecture**
   - Message queue (RabbitMQ/Kafka)
   - Async order processing
   - Email notifications

2. **Microservices Split**
   - Separate Order Service
   - Separate Product Catalog Service
   - API Gateway (Kong/Traefik)

3. **Advanced Caching**
   - Redis pub/sub for cache invalidation
   - Multi-level caching (L1: in-memory, L2: Redis)

4. **Observability**
   - Distributed tracing (Jaeger)
   - Metrics (Prometheus + Grafana)
   - APM (New Relic / Datadog)

---

## References

- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Standard Project Layout](https://github.com/golang-standards/project-layout)
- [Twelve-Factor App Methodology](https://12factor.net/)
- [PostgreSQL Best Practices](https://wiki.postgresql.org/wiki/Don%27t_Do_This)
