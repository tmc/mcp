package mcp

import (
	"testing"
)

// TestWithDispatcher tests the WithDispatcher server option
func TestWithDispatcher(t *testing.T) {
	dispatcher := NewDispatcher()

	// Create server with custom dispatcher
	server := NewServer("test", "1.0", WithDispatcher(dispatcher))

	// Verify the dispatcher was set
	if server.dispatch != dispatcher {
		t.Error("WithDispatcher option did not set the dispatcher correctly")
	}
}

// TestWithCapabilities tests the WithCapabilities server option
func TestWithCapabilities(t *testing.T) {
	capabilities := ServerCapabilities{
		Experimental: map[string]interface{}{
			"test": true,
		},
	}

	// Create server with custom capabilities
	server := NewServer("test", "1.0", WithCapabilities(capabilities))

	// Verify the capabilities were set
	if server.capabilities.Experimental == nil {
		t.Error("WithCapabilities option did not set capabilities correctly")
	}

	if val, ok := server.capabilities.Experimental["test"].(bool); !ok || !val {
		t.Error("WithCapabilities option did not set experimental capabilities correctly")
	}
}
