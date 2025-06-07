package mcp

import (
	"context"
	"testing"
	"time"
)

// TestClientUninitialized tests methods on uninitialized client
func TestClientUninitialized(t *testing.T) {
	client := &Client{}

	// All these should fail with "not initialized" error
	ctx := context.Background()

	_, err := client.ListTools(ctx, ListToolsRequest{})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for ListTools, got: %v", err)
	}

	_, err = client.CallTool(ctx, CallToolRequest{Name: "test"})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for CallTool, got: %v", err)
	}

	_, err = client.ListPrompts(ctx, ListPromptsRequest{})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for ListPrompts, got: %v", err)
	}

	_, err = client.GetPrompt(ctx, GetPromptRequest{Name: "test"})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for GetPrompt, got: %v", err)
	}

	_, err = client.ListResources(ctx, ListResourcesRequest{})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for ListResources, got: %v", err)
	}

	_, err = client.ReadResource(ctx, ReadResourceRequest{URI: "test"})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for ReadResource, got: %v", err)
	}

	_, err = client.ListResourceTemplates(ctx, ListResourceTemplatesRequest{})
	if err == nil || err.Error() != "client not initialized, call Initialize() first" {
		t.Errorf("Expected initialization error for ListResourceTemplates, got: %v", err)
	}
}

// TestClientNotificationHandler tests notification handler setup
func TestClientNotificationHandler(t *testing.T) {
	client := &Client{}

	// Test setting handler via option
	handler := func(n JSONRPCNotification) {
		// Handler implementation
	}

	WithNotificationHandler(handler)(client)

	// Verify handler is set
	client.notificationMu.RLock()
	if client.notifyHandler == nil {
		t.Error("Handler not set via option")
	}
	client.notificationMu.RUnlock()

	// Test OnNotification method
	called2 := false
	client.OnNotification(func(n JSONRPCNotification) {
		called2 = true
	})

	// Trigger the handler
	client.notificationMu.RLock()
	h := client.notifyHandler
	client.notificationMu.RUnlock()

	h(JSONRPCNotification{Method: "test"})

	if !called2 {
		t.Error("OnNotification handler not called")
	}
}

// TestClientInitializeEdgeCases tests edge cases for Initialize
func TestClientInitializeEdgeCases(t *testing.T) {
	// Test with nil connection
	t.Run("nil connection", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		client := &Client{
			conn: nil,
		}

		_, err := client.Initialize(ctx, InitializeRequest{
			ClientInfo: Implementation{
				Name:    "test",
				Version: "1.0",
			},
		})

		// Should fail due to nil connection
		if err == nil {
			t.Error("Expected error with nil connection")
		}
		if err.Error() != "client connection is not established" {
			t.Errorf("Expected 'client connection is not established' error, got: %v", err)
		}
	})
}
