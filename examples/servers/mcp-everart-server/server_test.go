package main

import (
	"testing"

	"log/slog"

	"github.com/tmc/mcp"
)

func TestEverArtServer(t *testing.T) {
	server := mcp.NewServer(ServerName, ServerVersion, mcp.WithTestLogger(t, slog.LevelDebug))
	registerTools(server)

	// Test that the server was created successfully
	if server == nil {
		t.Fatal("Failed to create server")
	}
}
