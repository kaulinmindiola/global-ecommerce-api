package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/metrics"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/middleware"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
)

// RouterDeps holds all application dependencies needed to wire the HTTP router.
type RouterDeps struct {
	// Services
	UserService     service.UserService
	AuthService     service.AuthService
	CurrencyService service.CurrencyService
	ProductService  service.ProductService
	OrderService    service.OrderService

	// Infrastructure
	DB      *pgxpool.Pool
	Redis   *redis.Client
	Metrics *metrics.Metrics

	// Build metadata
	Version    string
	BuildDate  string
	CommitHash string
}

// NewRouter builds and returns the fully configured Chi router.
func NewRouter(deps *RouterDeps) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware stack ──────────────────────────────────────────
	r.Use(chiMiddleware.RealIP)                       // 1. Trust X-Forwarded-For
	r.Use(middleware.RequestID)                       // 2. Attach trace ID
	r.Use(middleware.Logger)                          // 3. Structured request logging
	r.Use(middleware.Recovery)                        // 4. Catch panics, return 500 cleanly
	r.Use(middleware.PrometheusMetrics(deps.Metrics)) // 5. Prometheus metrics
	r.Use(chiMiddleware.Compress(5))                  // 6. gzip compression level 5
	r.Use(middleware.CORS)                            // 7. CORS headers

	// ── Handler instances ────────────────────────────────────────────────
	healthHandler := NewHealthHandler(
		deps.DB, deps.Redis, deps.Metrics,
		deps.Version, deps.BuildDate, deps.CommitHash,
	)
	authHandler := NewAuthHandler(deps.UserService, deps.AuthService)
	userHandler := NewUserHandler(deps.UserService)
	currencyHandler := NewCurrencyHandler(deps.CurrencyService)
	productHandler := NewProductHandler(deps.ProductService)
	orderHandler := NewOrderHandler(deps.OrderService)

	// ── Prometheus scrape endpoint ───────────────────────────────────────
	r.Handle("/metrics", promhttp.HandlerFor(
		deps.Metrics.Registry(),
		promhttp.HandlerOpts{EnableOpenMetrics: true},
	))

	// ── Swagger UI (API Documentation) ───────────────────────────────────

	// Redirigir la ruta base a la interfaz HTML para mejor DX
	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})

	// Servir el contrato OpenAPI como archivo estático
	r.Get("/swagger/doc.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi/openapi.yaml")
	})

	// Swagger UI consumiendo el YAML como fuente única de verdad
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.yaml"),
	))

	// ── API v1 prefix ────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		// ── Observability (public, no auth) ──────────────────────────────
		r.Get("/health", healthHandler.Health)
		r.Get("/version", healthHandler.Version)

		// ── Authentication (public) ──────────────────────────────────────
		r.Route("/auth", func(r chi.Router) {
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

		// ── Users ────────────────────────────────────────────────────────
		r.Route("/users", func(r chi.Router) {
			r.Use(middleware.Authenticate(deps.AuthService))
			r.Use(middleware.RateLimit(deps.Redis))

			r.Get("/me", userHandler.GetMe)
			r.With(middleware.ValidateBody).Put("/me", userHandler.UpdateMe)
			r.Delete("/me", userHandler.DeleteMe)
			r.Get("/{id}", userHandler.GetByID)
		})

		// ── Orders ───────────────────────────────────────────────────────
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
