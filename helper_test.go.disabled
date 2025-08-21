package mcp_test

import (
	"context"
	"testing"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/testing/mcptestutil"
)

// TestSimpleHelper tests basic helper functionality
func TestSimpleHelper(t *testing.T) {
	// Create a simple server
	server := mcp.NewServer("test-server", "1.0.0")

	ctx := context.Background()
	pair, err := mcptestutil.NewServerClientPair(t, ctx, server)
	if err != nil {
		t.Fatalf("Failed to create server/client pair: %v", err)
	}
	defer pair.Cleanup()

	// Simple check that client is initialized
	if pair.Client == nil {
		t.Error("Client is nil")
	}

	if pair.Server == nil {
		t.Error("Server is nil")
	}
}
