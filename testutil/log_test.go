package testutil_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/testutil"
)

// TestLogLevels demonstrates the logging behavior at different levels
func TestLogLevels(t *testing.T) {
	server := mcp.NewServer("log-test-server", "1.0.0",
		testutil.WithTestLogger(t, slog.LevelDebug),
	)
	
	// Create a simple tool that logs at different levels
	tool := mcp.Tool{
		Name:        "log_tester",
		Description: "Tests logging at different levels",
		InputSchema: []byte(`{"type": "object"}`),
	}
	
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Use a logger that goes through the test handler
		logger := testutil.TestLogger(t)
		logger.Debug("This is a DEBUG message - only shown with -v or MCP_TEST_DEBUG=1")
		logger.Info("This is an INFO message - shown by default")
		logger.Warn("This is a WARN message - shown by default")
		logger.Error("This is an ERROR message - always shown")

		return &mcp.CallToolResult{
			Content: []any{
				mcp.TextContent{
					Type: "text", 
					Text: "Logged at all levels",
				},
			},
		}, nil
	}
	
	if err := server.RegisterTool(tool, handler); err != nil {
		t.Fatal(err)
	}
	
	// Create connected pair
	ctx := context.Background()
	pair, err := testutil.NewServerClientPair(t, ctx, server)
	if err != nil {
		t.Fatal(err)
	}
	defer pair.Cleanup()
	
	// Call the tool to trigger logs
	result, err := pair.Client.CallTool(ctx, mcp.CallToolRequest{
		Name: "log_tester",
		Arguments: []byte(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	
	t.Logf("Tool result: %+v", result)
}

// TestLogConfiguration verifies different logging configurations
func TestLogConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		desc  string
	}{
		{
			name:  "debug_level",
			level: slog.LevelDebug,
			desc:  "All logs visible with -v or MCP_TEST_DEBUG=1",
		},
		{
			name:  "info_level",
			level: slog.LevelInfo,
			desc:  "INFO and above visible by default",
		},
		{
			name:  "warn_level",
			level: slog.LevelWarn,
			desc:  "Only WARN and ERROR visible",
		},
		{
			name:  "error_level",
			level: slog.LevelError,
			desc:  "Only ERROR visible",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing %s: %s", tt.name, tt.desc)
			
			server := mcp.NewServer("test-server", "1.0.0",
				testutil.WithTestLogger(t, tt.level),
			)
			
			// Just create the pair to test logger setup
			pair, err := testutil.NewServerClientPair(t, context.Background(), server)
			if err != nil {
				t.Fatal(err)
			}
			defer pair.Cleanup()
		})
	}
}