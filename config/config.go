// Package config provides centralised configuration management following the
// 12-factor app methodology (https://12factor.net/config).
//
// Load priority (highest wins):
//  1. OS environment variables  — production, Docker, Kubernetes, CI
//  2. .env file                 — developer convenience in local environments
//  3. config.yaml               — safe baseline defaults
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// ── Top-level Config struct ───────────────────────────────────────────────────

// Config is the single source of truth for all application settings.
type Config struct {
	App       AppConfig       `yaml:"app"`
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	Log       LogConfig       `yaml:"logging"` // Mapeado a "logging" del YAML
	CORS      CORSConfig      `yaml:"cors"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// AppConfig holds general application metadata.
type AppConfig struct {
	Name        string `yaml:"name"`
	Environment string `yaml:"environment"`
	Version     string `yaml:"version"`
	BuildDate   string `yaml:"build_date"`
	CommitHash  string `yaml:"commit_hash"`
}

// ServerConfig holds HTTP server parameters.
type ServerConfig struct {
	Port            string        `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// DatabaseConfig holds all PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string       `yaml:"host"`
	Port     int          `yaml:"port"`
	DBName   string       `yaml:"name"` // Mapeado a "name" del YAML
	User     string       `yaml:"user"`
	Password string       `yaml:"-"` // never in YAML
	SSLMode  string       `yaml:"ssl_mode"`
	Pool     DBPoolConfig `yaml:"pool"` // Estructura anidada para el pool
}

// DBPoolConfig holds go-pgx pool tuning parameters.
type DBPoolConfig struct {
	MaxConns          int32         `yaml:"max_conns"`
	MinConns          int32         `yaml:"min_conns"`
	MaxConnLifetime   time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `yaml:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
}

// DSN builds the PostgreSQL connection string.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string          `yaml:"host"`
	Port     string          `yaml:"port"`
	Password string          `yaml:"-"` // never in YAML
	DB       int             `yaml:"db"`
	Pool     RedisPoolConfig `yaml:"pool"`
}

// RedisPoolConfig holds go-redis pool tuning parameters.
type RedisPoolConfig struct {
	PoolSize        int           `yaml:"pool_size"`
	MinIdleConns    int           `yaml:"min_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"`
	DialTimeout     time.Duration `yaml:"dial_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
}

// Addr returns the Redis address in host:port format.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

// JWTConfig holds JWT signing and lifetime settings.
type JWTConfig struct {
	SecretKey       string        `yaml:"-"` // never in YAML
	AccessTokenTTL  time.Duration `yaml:"access_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_ttl"`
}

// LogConfig controls the structured logger behaviour.
type LogConfig struct {
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	AddSource bool   `yaml:"add_source"`
}

// CORSConfig holds CORS policy settings.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins"`
	MaxAge         string `yaml:"max_age"`
}

// AllowedOriginList parses the comma-separated AllowedOrigins into a slice.
func (c CORSConfig) AllowedOriginList() []string {
	var origins []string
	for _, o := range strings.Split(c.AllowedOrigins, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	RequestsPerWindow int           `yaml:"requests_per_window"`
	Window            time.Duration `yaml:"window"`
}

// ── Loader ────────────────────────────────────────────────────────────────────

// Load builds the Config by merging sources in priority order.
func Load() (*Config, error) {
	cfg, err := loadYAMLDefaults()
	if err != nil {
		return nil, fmt.Errorf("loading config.yaml defaults: %w", err)
	}

	_ = godotenv.Load()

	applyEnvOverrides(cfg)
	applyBuildMetadata(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func loadYAMLDefaults() (*Config, error) {
	cfg := &Config{}
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..")
	yamlPath := filepath.Join(projectRoot, "config", "config.yaml")

	data, err := os.ReadFile(yamlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config.yaml: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config.yaml: %w", err)
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	// App
	setString(&cfg.App.Environment, "APP_ENV")
	setString(&cfg.App.Name, "APP_NAME")

	// Server
	setString(&cfg.Server.Port, "APP_PORT")
	setDuration(&cfg.Server.ReadTimeout, "SERVER_READ_TIMEOUT")
	setDuration(&cfg.Server.WriteTimeout, "SERVER_WRITE_TIMEOUT")
	setDuration(&cfg.Server.IdleTimeout, "SERVER_IDLE_TIMEOUT")
	setDuration(&cfg.Server.ShutdownTimeout, "SERVER_SHUTDOWN_TIMEOUT")

	// Database (Ajustado para acceder a cfg.Database.Pool.*)
	setString(&cfg.Database.Host, "DATABASE_HOST")
	setInt(&cfg.Database.Port, "DATABASE_PORT")
	setString(&cfg.Database.User, "DATABASE_USER")
	setString(&cfg.Database.Password, "DATABASE_PASSWORD")
	setString(&cfg.Database.DBName, "DATABASE_NAME")
	setString(&cfg.Database.SSLMode, "DATABASE_SSL_MODE")
	setInt32(&cfg.Database.Pool.MaxConns, "DATABASE_MAX_CONNS")
	setInt32(&cfg.Database.Pool.MinConns, "DATABASE_MIN_CONNS")
	setDuration(&cfg.Database.Pool.MaxConnLifetime, "DATABASE_MAX_CONN_LIFETIME")
	setDuration(&cfg.Database.Pool.MaxConnIdleTime, "DATABASE_MAX_CONN_IDLE")
	setDuration(&cfg.Database.Pool.HealthCheckPeriod, "DATABASE_HEALTH_CHECK")

	// Redis
	setString(&cfg.Redis.Host, "REDIS_HOST")
	setString(&cfg.Redis.Port, "REDIS_PORT")
	setString(&cfg.Redis.Password, "REDIS_PASSWORD")
	setInt(&cfg.Redis.DB, "REDIS_DB")

	// JWT
	setString(&cfg.JWT.SecretKey, "JWT_SECRET")
	setDuration(&cfg.JWT.AccessTokenTTL, "JWT_ACCESS_TTL")
	setDuration(&cfg.JWT.RefreshTokenTTL, "JWT_REFRESH_TTL")

	// Log
	setString(&cfg.Log.Level, "LOG_LEVEL")
	setString(&cfg.Log.Format, "LOG_FORMAT")

	// CORS & Rate Limit
	setString(&cfg.CORS.AllowedOrigins, "CORS_ALLOWED_ORIGINS")
	setIntFromEnv(&cfg.RateLimit.RequestsPerWindow, "RATE_LIMIT_REQUESTS")
	setDuration(&cfg.RateLimit.Window, "RATE_LIMIT_WINDOW")
}

func applyBuildMetadata(cfg *Config) {
	setString(&cfg.App.Version, "APP_VERSION")
	setString(&cfg.App.BuildDate, "APP_BUILD_DATE")
	setString(&cfg.App.CommitHash, "APP_COMMIT_HASH")
}

func validate(cfg *Config) error {
	var missing []string

	if cfg.Database.Password == "" {
		missing = append(missing, "DATABASE_PASSWORD")
	}
	if cfg.Database.Host == "" {
		missing = append(missing, "DATABASE_HOST")
	}
	if cfg.Database.User == "" {
		missing = append(missing, "DATABASE_USER")
	}
	if cfg.Database.DBName == "" {
		missing = append(missing, "DATABASE_NAME")
	}
	if cfg.JWT.SecretKey == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.Server.Port == "" {
		missing = append(missing, "APP_PORT")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// ── Environment helpers & Masking ─────────────────────────────────────────────

func (c *Config) IsDevelopment() bool { return c.App.Environment == "development" }
func (c *Config) IsProduction() bool  { return c.App.Environment == "production" }
func (c *Config) IsStaging() bool     { return c.App.Environment == "staging" }

// Redacted returns a safe copy for logging.
func (c *Config) Redacted() map[string]any {
	return map[string]any{
		"app": map[string]any{
			"environment": c.App.Environment,
			"version":     c.App.Version,
			"commit":      c.App.CommitHash,
		},
		"database": map[string]any{
			"host":     c.Database.Host,
			"port":     c.Database.Port,
			"name":     c.Database.DBName,
			"user":     c.Database.User,
			"password": "[REDACTED]",
			"pool": map[string]any{
				"max_conns": c.Database.Pool.MaxConns,
			},
		},
		"redis": map[string]any{
			"addr":     c.Redis.Addr(),
			"db":       c.Redis.DB,
			"password": "[REDACTED]",
		},
		"jwt": map[string]any{
			"secret": "[REDACTED]",
		},
		"log": map[string]any{
			"level":  c.Log.Level,
			"format": c.Log.Format,
		},
	}
}

// ── Low-level env setters ─────────────────────────────────────────────────────

func setString(dest *string, key string) {
	if v := os.Getenv(key); v != "" {
		*dest = v
	}
}

func setInt(dest *int, key string) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dest = n
		}
	}
}

func setIntFromEnv(dest *int, key string) { setInt(dest, key) }

func setInt32(dest *int32, key string) {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*dest = int32(n)
		}
	}
}

func setDuration(dest *time.Duration, key string) {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*dest = d
		}
	}
}
