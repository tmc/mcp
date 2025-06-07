package main

import (
	"context"
	"encoding/json"
	"testing"

	"log/slog"

	"github.com/tmc/mcp"
)

// TestToolsIntegration tests the tools registered in the everything server
func TestToolsIntegration(t *testing.T) {
	// Create a new server instance
	server := mcp.NewServer("test-server", "1.0.0",
		mcp.WithServerInstructions("Test server for everything"),
	)

	// Register tools using the same function from main.go
	registerTools(server)

	// Test echo tool - note that the current implementation uses "message" parameter
	t.Run("EchoTool", func(t *testing.T) {
		// Since we can't directly call handlers, we'll test the tool registration
		found := false
		server.mu.RLock()
		for name, toolDef := range server.tools {
			if name == "echo" {
				found = true
				// Verify the tool definition
				if toolDef.tool.Name != "echo" {
					t.Errorf("Expected tool name 'echo', got %s", toolDef.tool.Name)
				}
				if toolDef.tool.Description != "Echo the input" {
					t.Errorf("Expected description 'Echo the input', got %s", toolDef.tool.Description)
				}

				// Test the tool handler directly
				req := mcp.CallToolRequest{
					Name:      "echo",
					Arguments: json.RawMessage(`{"message": "test message"}`),
				}

				result, err := toolDef.handler(context.Background(), req)
				if err != nil {
					t.Errorf("Echo tool handler failed: %v", err)
				} else {
					// Check the result content
					if content, ok := result.Content.([]interface{}); ok && len(content) > 0 {
						if item, ok := content[0].(map[string]interface{}); ok {
							if text, ok := item["text"].(string); ok {
								expectedText := "Echo: test message"
								if text != expectedText {
									t.Errorf("Expected echo text %q, got %q", expectedText, text)
								}
							} else {
								t.Error("No text field in result")
							}
						} else {
							t.Error("Invalid content format")
						}
					} else {
						t.Error("No content in result")
					}
				}
			}
		}
		server.mu.RUnlock()

		if !found {
			t.Error("Echo tool not found in registered tools")
		}
	})

	// Test current_time tool
	t.Run("CurrentTimeTool", func(t *testing.T) {
		found := false
		server.mu.RLock()
		for name, toolDef := range server.tools {
			if name == "current_time" {
				found = true
				// Test with valid timezone
				req := mcp.CallToolRequest{
					Name:      "current_time",
					Arguments: json.RawMessage(`{"timezone": "UTC"}`),
				}

				result, err := toolDef.handler(context.Background(), req)
				if err != nil {
					t.Errorf("Time tool handler failed: %v", err)
				} else {
					// Check that we got content
					if content, ok := result.Content.([]interface{}); !ok || len(content) == 0 {
						t.Error("Expected content in result")
					}
				}
			}
		}
		server.mu.RUnlock()

		if !found {
			t.Error("current_time tool not found in registered tools")
		}
	})

	// Test random tool
	t.Run("RandomTool", func(t *testing.T) {
		found := false
		server.mu.RLock()
		for name, toolDef := range server.tools {
			if name == "random" {
				found = true
				// Test with valid range
				req := mcp.CallToolRequest{
					Name:      "random",
					Arguments: json.RawMessage(`{"min": 10, "max": 20}`),
				}

				result, err := toolDef.handler(context.Background(), req)
				if err != nil {
					t.Errorf("Random tool handler failed: %v", err)
				} else {
					// Check that we got content with a number in the expected format
					if content, ok := result.Content.([]interface{}); ok && len(content) > 0 {
						if item, ok := content[0].(map[string]interface{}); ok {
							if _, hasText := item["text"]; !hasText {
								t.Error("No text field in result")
							}
						}
					}
				}
			}
		}
		server.mu.RUnlock()

		if !found {
			t.Error("random tool not found in registered tools")
		}
	})
}

// TestToolsListHandler tests the tools/list functionality
func TestToolsListHandler(t *testing.T) {
	// Create a new server instance
	server := mcp.NewServer("test-server", "1.0.0", WithTestLogger(t, slog.LevelDebug))
	registerTools(server)

	// Count the registered tools
	server.mu.RLock()
	toolCount := len(server.tools)
	server.mu.RUnlock()

	expectedCount := 3 // echo, current_time, random
	if toolCount != expectedCount {
		t.Errorf("Expected %d tools, got %d", expectedCount, toolCount)
	}
}
