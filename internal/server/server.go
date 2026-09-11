package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps the standard library HTTP server and adds graceful shutdown.
// Abstracting net/http.Server here keeps main.go clean and makes the server
// independently testable.
type Server struct {
	httpServer *http.Server
}

// New creates a new Server with production-grade timeout settings.
// All timeouts are configured from the application Config to avoid magic numbers.
func New(handler http.Handler, port string, readTimeout, writeTimeout, idleTimeout time.Duration) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: handler,

			// ReadTimeout: max time to read the full request (headers + body).
			// Prevents slow-loris attacks where clients send headers very slowly.
			ReadTimeout: readTimeout,

			// WriteTimeout: max time to write the full response.
			// Prevents goroutine leaks from slow clients.
			WriteTimeout: writeTimeout,

			// IdleTimeout: max time a keep-alive connection stays open between requests.
			IdleTimeout: idleTimeout,
		},
	}
}

// Start begins listening for HTTP connections on the configured port.
// This method blocks until the server stops (either from an error or via Shutdown).
// It returns http.ErrServerClosed on clean shutdown, which callers should treat as nil.
func (s *Server) Start() error {
	slog.Info("HTTP server starting", "addr", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server error: %w", err)
	}

	return nil
}

// Shutdown gracefully drains in-flight requests before closing.
//
// Shutdown sequence:
//  1. Stop accepting new connections immediately.
//  2. Wait for active request handlers to finish (up to shutdownTimeout).
//  3. Close all idle keep-alive connections.
//  4. Return once all connections are drained or the timeout expires.
//
// The shutdownTimeout should be long enough for the slowest expected
// request (e.g., a complex order creation) to complete — 30s is a safe default.
func (s *Server) Shutdown(shutdownTimeout time.Duration) error {
	slog.Info("HTTP server shutting down gracefully",
		"timeout", shutdownTimeout,
	)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info("HTTP server shutdown complete")
	return nil
}

// Addr returns the address the server is listening on (e.g. ":8080").
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
