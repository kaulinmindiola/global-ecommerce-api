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

	// Infrastructure
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
// Route layout:
//
//	/api/v1/health            — public
//	/api/v1/version           — public
//	/api/v1/auth/*            — public
//	/api/v1/currencies/*      — public (read-only)
//	/api/v1/products/*        — GET public, POST/PUT/DELETE require auth
//	/api/v1/users/*           — requires auth
//	/api/v1/orders/*          — requires auth
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware stack ──────────────────────────────────────────
	// Applied to every request regardless of route.
	r.Use(chiMiddleware.RealIP)      // Trust X-Forwarded-For
	r.Use(middleware.RequestID)      // Attach trace ID to context + response header
	r.Use(middleware.Logger)         // Structured request logging
	r.Use(middleware.Recoverer)      // Catch panics, return 500
	r.Use(chiMiddleware.Compress(5)) // gzip compression level 5
	r.Use(middleware.CORS)           // CORS headers

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
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
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
		// Write endpoints (POST, PUT, DELETE) require a valid JWT.
		r.Route("/products", func(r chi.Router) {
			r.Get("/", productHandler.ListProducts)
			r.Get("/{id}", productHandler.GetProduct)

			// Write routes — protected by JWT middleware.
			r.Group(func(r chi.Router) {
				r.Use(middleware.Authenticate(deps.AuthService))
				r.Post("/", productHandler.CreateProduct)
				r.Put("/{id}", productHandler.UpdateProduct)
				r.Delete("/{id}", productHandler.DeleteProduct)
			})
		})

		// ── Users (all routes require auth) ──────────────────────────────
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.AuthService))
			r.Get("/me", userHandler.GetMe)
			r.Put("/me", userHandler.UpdateMe)
			r.Delete("/me", userHandler.DeleteMe)
			r.Get("/{id}", userHandler.GetByID) // admin endpoint
		})

		// ── Orders (all routes require auth) ─────────────────────────────
		r.Route("/orders", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.AuthService))
			r.Post("/", orderHandler.CreateOrder)
			r.Get("/", orderHandler.ListOrders)
			r.Get("/number/{orderNumber}", orderHandler.GetOrderByNumber)
			r.Get("/{id}", orderHandler.GetOrder)
			r.Put("/{id}/status", orderHandler.UpdateOrderStatus)
		})
	})

	return r
}
