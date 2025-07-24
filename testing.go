package mcp

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// Logger represents the interface needed for logging in tests
type Logger interface {
	Logf(format string, args ...interface{})
}

// testLogHandler implements slog.Handler that redirects to t.Log
type testLogHandler struct {
	logger  Logger
	level   slog.Level
	verbose bool
}

func (h *testLogHandler) Enabled(_ context.Context, level slog.Level) bool {
	// Check if running in short mode - be much quieter
	if testing.Short() && !h.verbose {
		return level >= slog.LevelError
	}

	// Without -v: Show WARN and above to reduce test output clutter
	// With -v: Show INFO and above
	// With MCP_TEST_DEBUG=1: Show DEBUG and above
	if !h.verbose {
		return level >= slog.LevelWarn
	}
	if testing.Verbose() {
		return level >= slog.LevelInfo
	}
	return level >= h.level
}

func (h *testLogHandler) Handle(_ context.Context, record slog.Record) error {
	// In short mode and non-verbose, don't output anything to avoid cluttering test output
	if testing.Short() && !h.verbose {
		return nil
	}

	if h.logger != nil {
		h.logger.Logf("[%s] %s", record.Level, record.Message)
	}
	return nil
}

func (h *testLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *testLogHandler) WithGroup(name string) slog.Handler {
	return h
}

// WithTestLogger creates a server option that redirects logs to t.Log
// Default behavior:
// - Without -v: Shows INFO and above
// - With -v: Shows DEBUG and above
// - With MCP_TEST_DEBUG=1: Always shows DEBUG and above
func WithTestLogger(logger Logger, level slog.Level) ServerOption {
	// Check for debug environment variable
	verbose := testing.Verbose() || os.Getenv("MCP_TEST_DEBUG") == "1"

	return WithLogger(slog.New(&testLogHandler{
		logger:  logger,
		level:   level,
		verbose: verbose,
	}))
}
