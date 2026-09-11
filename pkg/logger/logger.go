// Package logger provides a structured, leveled logger for the application.
//
// It wraps the standard library's log/slog with a fluent API and JSON output
// suitable for production log aggregation tools (Datadog, CloudWatch, Loki).
//
// Usage:
//
//	log := logger.New(logger.Config{Level: "info", Format: "json"})
//	log.Info("order created", "order_id", id, "total", 99.99)
//	log.WithField("user_id", userID).Error("payment failed", "reason", err)
package logger

import (
	"context"
	"log/slog"
	"os"
)

// contextKey is the unexported key type for logger values stored in context.
type contextKey struct{}

// Config holds all logger configuration parameters.
type Config struct {
	// Level controls the minimum log severity: debug, info, warn, error.
	Level string
	// Format controls output encoding: json (production) or text (development).
	Format string
	// AddSource includes the file name and line number in every log entry.
	// Enable in debug mode only — adds overhead in production.
	AddSource bool
	// ServiceName is added to every log entry as a "service" field.
	// Useful when multiple services ship logs to the same aggregator.
	ServiceName string
}

// Logger is the application-level structured logger.
// It wraps slog.Logger and adds convenience methods and context propagation.
type Logger struct {
	inner *slog.Logger
}

// New creates a new Logger from the given Config.
// This should be called once in main() and the result injected throughout the app.
func New(cfg Config) *Logger {
	level := parseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	inner := slog.New(handler)

	// Attach the service name as a permanent field on every log entry.
	// This is critical in multi-service environments for log filtering.
	if cfg.ServiceName != "" {
		inner = inner.With("service", cfg.ServiceName)
	}

	// Set as the global default so existing slog.Info() calls also use this config.
	slog.SetDefault(inner)

	return &Logger{inner: inner}
}

// ── Core log methods ─────────────────────────────────────────────────────────

// Debug logs a message at DEBUG level with optional key-value pairs.
// Use for diagnostic information useful during development.
func (l *Logger) Debug(msg string, args ...any) {
	l.inner.Debug(msg, args...)
}

// Info logs a message at INFO level with optional key-value pairs.
// Use for normal application events (server started, request processed).
func (l *Logger) Info(msg string, args ...any) {
	l.inner.Info(msg, args...)
}

// Warn logs a message at WARN level with optional key-value pairs.
// Use for recoverable issues that should be investigated (retry succeeded, deprecated usage).
func (l *Logger) Warn(msg string, args ...any) {
	l.inner.Warn(msg, args...)
}

// Error logs a message at ERROR level with optional key-value pairs.
// Use for failures that affect a user request but do not crash the server.
func (l *Logger) Error(msg string, args ...any) {
	l.inner.Error(msg, args...)
}

// ── Context-aware logging ────────────────────────────────────────────────────

// FromContext retrieves the logger stored in a context, or returns the
// global default logger if none was set.
// This enables request-scoped logging with fields like request_id and user_id
// automatically attached to every log entry within a request's lifecycle.
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok && l != nil {
		return l
	}
	return &Logger{inner: slog.Default()}
}

// WithContext stores the logger in a context for propagation through
// the request pipeline. Called by the logging middleware.
func WithContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// ── Field builders ───────────────────────────────────────────────────────────

// WithField returns a new Logger with the given key-value pair permanently
// attached to every subsequent log entry. Ideal for request-scoped fields.
//
// Example:
//
//	requestLogger := log.WithField("request_id", reqID).WithField("user_id", userID)
//	requestLogger.Info("processing order")  // includes request_id and user_id
func (l *Logger) WithField(key string, value any) *Logger {
	return &Logger{inner: l.inner.With(key, value)}
}

// WithFields returns a new Logger with multiple key-value pairs attached.
// args must be alternating key, value pairs (same convention as slog).
func (l *Logger) WithFields(args ...any) *Logger {
	return &Logger{inner: l.inner.With(args...)}
}

// WithError returns a new Logger with an "error" field attached.
// This is a convenience wrapper since error logging is extremely common.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{inner: l.inner.With("error", err.Error())}
}

// WithRequestID returns a new Logger with the "request_id" field attached.
// Called by the logging middleware to propagate the trace ID.
func (l *Logger) WithRequestID(id string) *Logger {
	return &Logger{inner: l.inner.With("request_id", id)}
}

// ── Internal helpers ─────────────────────────────────────────────────────────

// parseLevel converts a string level name to a slog.Level.
// Falls back to INFO for unrecognised values.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
