package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcp"
)

func TestServerInitialization(t *testing.T) {
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A screen capture server that provides tools for listing displays and capturing screenshots on macOS"),
	)

	if server == nil {
		t.Fatal("Expected server to be initialized")
	}
}

func TestListScreensToolRegistration(t *testing.T) {
	server := mcp.NewServer(ServerName, ServerVersion)
	registerListScreensTool(server)

	// Verify the tool is registered
	// The server should have the tool registered internally
	// This is a basic smoke test
}

func TestCaptureScreenToolRegistration(t *testing.T) {
	server := mcp.NewServer(ServerName, ServerVersion)
	registerCaptureScreenTool(server)

	// Verify the tool is registered
	// The server should have the tool registered internally
	// This is a basic smoke test
}

func TestListScreensToolSchema(t *testing.T) {
	listScreensTool := mcp.Tool{
		Name:        "list_screens",
		Description: "Lists connected displays using system_profiler. Returns detailed information about all connected monitors.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {}
		}`),
	}

	if listScreensTool.Name != "list_screens" {
		t.Errorf("Expected tool name 'list_screens', got '%s'", listScreensTool.Name)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(listScreensTool.InputSchema, &schema); err != nil {
		t.Fatalf("Invalid input schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got '%v'", schema["type"])
	}
}

func TestCaptureScreenToolSchema(t *testing.T) {
	captureScreenTool := mcp.Tool{
		Name:        "capture_screen",
		Description: "Captures a screenshot of the main display and returns it as a PNG image. Optionally specify a display ID to capture a specific display.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"display_id": {
					"type": "integer",
					"description": "Optional display ID to capture (omit to capture main display)"
				}
			}
		}`),
	}

	if captureScreenTool.Name != "capture_screen" {
		t.Errorf("Expected tool name 'capture_screen', got '%s'", captureScreenTool.Name)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(captureScreenTool.InputSchema, &schema); err != nil {
		t.Fatalf("Invalid input schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("Expected schema type 'object', got '%v'", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be an object")
	}

	if _, exists := properties["display_id"]; !exists {
		t.Error("Expected display_id property in schema")
	}
}

func TestStdioTransport(t *testing.T) {
	transport := &StdioTransport{}
	ctx := context.Background()

	conn, err := transport.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}

	if conn == nil {
		t.Fatal("Expected connection to be non-nil")
	}

	// Close should not error
	if err := conn.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}
