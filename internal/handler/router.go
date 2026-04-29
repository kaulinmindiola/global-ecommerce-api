package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/middleware"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
	"github.com/redis/go-redis/v9"
)

// RouterDeps holds all application dependencies needed to wire the HTTP router.
// Using a single struct keeps NewRouter's signature clean and extensible.
type RouterDeps struct {
	// Services
	UserService     service.UserService
	AuthService     service.AuthService
	CurrencyService service.CurrencyService
	ProductService  service.ProductService
	OrderService    service.OrderService

	// Infrastructure — passed to health handler and rate limiter
	DB    *pgxpool.Pool
	Redis *redis.Client

	// Build metadata — injected via ldflags in production
	Version    string
	BuildDate  string
	CommitHash string
}

// NewRouter builds and returns the fully configured Chi router.
// Route groups and middleware are applied here — nowhere else.
//
// Middleware stack order (top = outermost, applied first):
//  1. RealIP   → Extracts true client IP (Critical for RateLimiter behind proxies)
//  2. RequestID→ Generates trace ID before anything else logs
//  3. Logger   → Logs request with IP and trace ID
//  4. Recoverer→ Catches panics in all downstream code, returns 500
//  5. Compress → Compresses responses before they reach the client
//  6. CORS     → Sets headers before any handler can short-circuit
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware stack ──────────────────────────────────────────
	// Applied to every request regardless of route. Order is CRITICAL.
	r.Use(chiMiddleware.RealIP)      // 1. Trust X-Forwarded-For
	r.Use(middleware.RequestID)      // 2. Attach trace ID
	r.Use(middleware.Logger)         // 3. Structured request logging
	r.Use(middleware.Recovery)       // 4. Catch panics, return 500 cleanly
	r.Use(chiMiddleware.Compress(5)) // 5. gzip compression level 5
	r.Use(middleware.CORS)           // 6. CORS headers

	// ── Handler instances ────────────────────────────────────────────────
	healthHandler := NewHealthHandler(deps.DB, deps.Redis, deps.Version, deps.BuildDate, deps.CommitHash)
	authHandler := NewAuthHandler(deps.UserService, deps.AuthService)
	userHandler := NewUserHandler(deps.UserService)
	currencyHandler := NewCurrencyHandler(deps.CurrencyService)
	productHandler := NewProductHandler(deps.ProductService)
	orderHandler := NewOrderHandler(deps.OrderService)

	// ── API v1 prefix ────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {

		// ── Health & Version (public, no auth) ───────────────────────────
		r.Get("/health", healthHandler.Health)
		r.Get("/version", healthHandler.Version)

		// ── Authentication (public) ──────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
			// ValidateBody applied strictly to endpoints expecting JSON payloads
			r.With(middleware.ValidateBody).Post("/register", authHandler.Register)
			r.With(middleware.ValidateBody).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.RefreshToken)
		})

		// ── Currencies (public, read-only) ───────────────────────────────
		r.Route("/currencies", func(r chi.Router) {
			r.Get("/", currencyHandler.ListCurrencies)
			r.Get("/convert", currencyHandler.ConvertCurrency)
		})

		// ── Products ─────────────────────────────────────────────────────
		// GET endpoints are public — browsing does not require authentication.
		// Write endpoints require valid JWT and Rate Limiting.
		r.Route("/products", func(r chi.Router) {
			r.Get("/", productHandler.ListProducts)
			r.Get("/{id}", productHandler.GetProduct)

			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.AuthService))
				r.Use(middleware.RateLimit(deps.Redis))

				r.With(middleware.ValidateBody).Post("/", productHandler.CreateProduct)
				r.With(middleware.ValidateBody).Put("/{id}", productHandler.UpdateProduct)
				r.Delete("/{id}", productHandler.DeleteProduct)
			})
		})

		// ── Users (all routes require auth + rate limit) ─────────────────
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.AuthService))
			r.Use(middleware.RateLimit(deps.Redis))

			r.Get("/me", userHandler.GetMe)
			r.With(middleware.ValidateBody).Put("/me", userHandler.UpdateMe)
			r.Delete("/me", userHandler.DeleteMe)
			r.Get("/{id}", userHandler.GetByID) // admin endpoint
		})

		// ── Orders (all routes require auth + rate limit) ────────────────
		r.Route("/orders", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.AuthService))
			r.Use(middleware.RateLimit(deps.Redis))

			r.With(middleware.ValidateBody).Post("/", orderHandler.CreateOrder)
			r.Get("/", orderHandler.ListOrders)
			r.Get("/number/{orderNumber}", orderHandler.GetOrderByNumber)
			r.Get("/{id}", orderHandler.GetOrder)
			r.With(middleware.ValidateBody).Put("/{id}/status", orderHandler.UpdateOrderStatus)
		})
	})

	return r
}
