package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the single source of truth for all application settings.
// It is loaded once at startup from environment variables and injected
// throughout the dependency graph — never read from os.Getenv again after main().
//
// Follows 12-factor app principle III: store config in the environment.
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Log      LogConfig
}

// AppConfig holds general application metadata.
type AppConfig struct {
	// Environment is one of: development, staging, production.
	Environment string
	// Version is injected at build time via ldflags.
	Version    string
	BuildDate  string
	CommitHash string
}

// ServerConfig holds HTTP server tuning parameters.
type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds all PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string

	// Connection pool tuning.
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

// DSN builds the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection parameters.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Addr returns the Redis address in host:port format.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// JWTConfig holds JWT signing and expiration settings.
type JWTConfig struct {
	SecretKey       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

// LogConfig controls the structured logger output.
type LogConfig struct {
	// Level is one of: debug, info, warn, error.
	Level string
	// Format is one of: json, text. Use json in production.
	Format string
}

// Load reads all configuration from environment variables.
// Returns an error immediately if any required variable is missing —
// fail-fast at startup is better than a runtime panic under load.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Environment: envString("APP_ENV", "development"),
			Version:     envString("APP_VERSION", "dev"),
			BuildDate:   envString("APP_BUILD_DATE", "unknown"),
			CommitHash:  envString("APP_COMMIT_HASH", "unknown"),
		},

		Server: ServerConfig{
			Port:            envString("APP_PORT", "8080"),
			ReadTimeout:     envDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    envDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     envDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: envDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},

		Redis: RedisConfig{
			Host:     envString("REDIS_HOST", "localhost"),
			Port:     envString("REDIS_PORT", "6379"),
			Password: envString("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},

		JWT: JWTConfig{
			AccessTokenTTL:  envDuration("JWT_ACCESS_TTL", 24*time.Hour),
			RefreshTokenTTL: envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},

		Log: LogConfig{
			Level:  envString("LOG_LEVEL", "info"),
			Format: envString("LOG_FORMAT", "json"),
		},
	}

	// ── Required fields — fail fast ──────────────────────────────────────

	dbHost, err := requireEnv("DATABASE_HOST")
	if err != nil {
		return nil, err
	}
	dbUser, err := requireEnv("DATABASE_USER")
	if err != nil {
		return nil, err
	}
	dbPassword, err := requireEnv("DATABASE_PASSWORD")
	if err != nil {
		return nil, err
	}
	dbName, err := requireEnv("DATABASE_NAME")
	if err != nil {
		return nil, err
	}
	jwtSecret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}

	cfg.Database = DatabaseConfig{
		Host:              dbHost,
		Port:              envInt("DATABASE_PORT", 5432),
		User:              dbUser,
		Password:          dbPassword,
		DBName:            dbName,
		SSLMode:           envString("DATABASE_SSL_MODE", "disable"),
		MaxConns:          int32(envInt("DATABASE_MAX_CONNS", 25)),
		MinConns:          int32(envInt("DATABASE_MIN_CONNS", 5)),
		MaxConnLifetime:   envDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute),
		MaxConnIdleTime:   envDuration("DATABASE_MAX_CONN_IDLE", 5*time.Minute),
		HealthCheckPeriod: envDuration("DATABASE_HEALTH_CHECK", 1*time.Minute),
	}

	cfg.JWT.SecretKey = jwtSecret

	return cfg, nil
}

// IsDevelopment returns true when running in a local development environment.
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction returns true when running in a production environment.
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// ── Environment variable helpers ─────────────────────────────────────────────

func requireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return val, nil
}

func envString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envInt(key string, defaultVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return val
}

func envDuration(key string, defaultVal time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultVal
	}
	return d
}
