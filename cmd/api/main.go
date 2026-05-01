// @title           Global E-commerce Microservices API
// @version         1.0.0
// @description     A production-grade RESTful API for international e-commerce platforms with multi-currency support, timezone-aware operations, and enterprise-level architecture.
// @termsOfService  https://github.com/kaulinmindiola/global-ecommerce-api

// @contact.name    Kaulin Mindiola
// @contact.email   kaulinmindiola@gmail.com
// @contact.url     https://github.com/kaulinmindiola

// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT

// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https

// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     JWT Authorization header using the Bearer scheme. Example: "Bearer {token}"
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/kaulinmindiola/global-ecommerce-api/api/docs"
	"github.com/kaulinmindiola/global-ecommerce-api/config"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/handler"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/infrastructure"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/metrics" // <-- NUEVA IMPORTACIÓN
	repoPostgres "github.com/kaulinmindiola/global-ecommerce-api/internal/repository/postgres"
	repoRedis "github.com/kaulinmindiola/global-ecommerce-api/internal/repository/redis"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/server"
	"github.com/kaulinmindiola/global-ecommerce-api/internal/service"
)

// Build-time variables injected via ldflags.
// Example: go build -ldflags="-X main.version=1.0.0 -X main.buildDate=2025-01-01"
var (
	version    = "dev"
	buildDate  = "unknown"
	commitHash = "unknown"
)

func main() {
	// ── Step 0: Bootstrap structured logger ──────────────────────────────
	// JSON logging in production, human-readable text in development.
	// Logger is configured before everything else so startup errors are captured.
	setupLogger("info", "json")

	slog.Info("Global E-commerce API starting",
		"version", version,
		"build_date", buildDate,
		"commit", commitHash,
	)

	// ── Step 1: Load environment variables ───────────────────────────────
	// godotenv loads .env file in development. In production (Docker/K8s),
	// env vars are injected directly and godotenv is a no-op.
	if err := godotenv.Load(); err != nil {
		slog.Debug(".env file not found, reading from environment directly")
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Re-configure logger with the values from config (may override defaults).
	setupLogger(cfg.Log.Level, cfg.Log.Format)

	slog.Info("Configuration loaded",
		"environment", cfg.App.Environment,
		"port", cfg.Server.Port,
	)

	// ── Step 2: Connect infrastructure ───────────────────────────────────
	// Context with a reasonable startup timeout prevents hanging forever
	// if the database or Redis is unreachable.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ReadTimeout*3)
	defer cancel()

	db, err := infrastructure.ConnectPostgres(ctx, cfg.Database)
	if err != nil {
		slog.Error("PostgreSQL connection failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		db.Close()
		slog.Info("PostgreSQL connection pool closed")
	}()

	redisClient, err := infrastructure.ConnectRedis(ctx, cfg.Redis)
	if err != nil {
		slog.Error("Redis connection failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = redisClient.Close()
		slog.Info("Redis connection closed")
	}()

	// ── Step 3: Initialise repository layer ──────────────────────────────
	// Repositories are pure data-access adapters — no business logic here.
	cacheRepo := repoRedis.NewCacheRepository(redisClient)
	userRepo := repoPostgres.NewUserRepository(db)
	productRepo := repoPostgres.NewProductRepository(db)
	currencyRepo := repoPostgres.NewCurrencyRepository(db)
	orderRepo := repoPostgres.NewOrderRepository(db)

	// ── Step 4: Initialise service layer ─────────────────────────────────
	// Services encapsulate all business logic and depend only on repository
	// interfaces — never on concrete implementations directly.
	authSvc := service.NewAuthService(cfg.JWT.SecretKey, cacheRepo)
	currencySvc := service.NewCurrencyService(currencyRepo, cacheRepo)
	userSvc := service.NewUserService(userRepo, currencyRepo, cacheRepo, currencySvc)
	productSvc := service.NewProductService(productRepo, currencyRepo, cacheRepo, currencySvc)
	orderSvc := service.NewOrderService(orderRepo, productRepo, userRepo, currencySvc, cacheRepo)

	appMetrics := metrics.New()

	// ── Step 5: Build the HTTP router ────────────────────────────────────
	// NewRouter wires all handlers and middleware groups.
	// All dependencies are injected here — no globals anywhere.
	router := handler.NewRouter(handler.RouterDeps{
		UserService:     userSvc,
		AuthService:     authSvc,
		CurrencyService: currencySvc,
		ProductService:  productSvc,
		OrderService:    orderSvc,
		DB:              db,
		Redis:           redisClient,
		Metrics:         appMetrics,
		Version:         cfg.App.Version,
		BuildDate:       cfg.App.BuildDate,
		CommitHash:      cfg.App.CommitHash,
	})

	// ── Step 6: Create and start HTTP server ─────────────────────────────
	srv := server.New(
		router,
		cfg.Server.Port,
		cfg.Server.ReadTimeout,
		cfg.Server.WriteTimeout,
		cfg.Server.IdleTimeout,
	)

	// Run the server in a goroutine so the main goroutine can listen for
	// OS signals concurrently without blocking.
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("HTTP server listening", "addr", srv.Addr())
		serverErrors <- srv.Start()
	}()

	// ── Step 7: Graceful shutdown ─────────────────────────────────────────
	// Listen for SIGINT (Ctrl+C) or SIGTERM (Docker/Kubernetes stop signal).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block here until either the server errors or a shutdown signal arrives.
	select {
	case err := <-serverErrors:
		slog.Error("Server error", "error", err)
		os.Exit(1)

	case sig := <-quit:
		slog.Info("Shutdown signal received", "signal", sig.String())

		// Give in-flight requests time to finish before closing connections.
		if err := srv.Shutdown(cfg.Server.ShutdownTimeout); err != nil {
			slog.Error("Graceful shutdown failed", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("Server exited cleanly")
}

// setupLogger configures the global slog logger.
//
// JSON format is used in production for log aggregation tools (Datadog, CloudWatch).
// Text format is used in development for human readability.
func setupLogger(level, format string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: logLevel == slog.LevelDebug, // include file:line only in debug mode
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}
