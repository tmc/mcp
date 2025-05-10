package mcp

import (
	"log/slog"
	"os"
)

// ServerOption configures a Server
type ServerOption func(*Server)

// WithRateLimiting configures custom rate limiting for the server
func WithRateLimiting(cfg RateLimitConfig) ServerOption {
	return func(s *Server) {
		s.limiter = NewRateLimiter(cfg)
	}
}

// WithDispatcher configures a custom notification dispatcher
func WithDispatcher(d *Dispatcher) ServerOption {
	return func(s *Server) {
		s.dispatch = d
	}
}

// WithCapabilities configures initial server capabilities
func WithCapabilities(caps ServerCapabilities) ServerOption {
	return func(s *Server) {
		s.capabilities = caps
	}
}

// WithLogger configures a custom logger for the server
func WithLogger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithLogLevel sets the log level for the server's logger
func WithLogLevel(level slog.Level) ServerOption {
	return func(s *Server) {
		if s.logger == nil {
			// If logger is not set, create a new one with the specified level
			opts := &slog.HandlerOptions{Level: level}
			s.logger = slog.New(slog.NewTextHandler(os.Stderr, opts))
			return
		}

		// Create a new handler with the specified level
		s.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	}
}
