package mcpmiddleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Log returns middleware that records method names, elapsed time, and errors.
// If logger is nil, slog.Default is used.
func Log(logger *slog.Logger) mcp.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			result, err := next(ctx, method, req)
			attrs := []slog.Attr{
				slog.String("method", method),
				slog.Duration("duration", time.Since(start)),
			}
			if err != nil {
				attrs = append(attrs, slog.Any("error", err))
			}
			logger.LogAttrs(ctx, slog.LevelInfo, "mcp method", attrs...)
			return result, err
		}
	}
}
