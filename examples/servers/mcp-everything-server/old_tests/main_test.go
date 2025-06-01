package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/tmc/mcp"
)

// TestRegisterTools tests that all expected tools are registered
func TestRegisterTools(t *testing.T) {
	server := mcp.NewServer("test-server", "1.0.0")
	registerTools(server)

	// Get the registered tools
	toolsHandler := server.GetToolsHandler()
	if toolsHandler == nil {
		t.Fatal("No tools handler found")
	}

	result, err := toolsHandler(context.Background(), mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	// Check that we have the expected tools
	expectedTools := []string{"current_time", "echo", "random"}
	toolMap := make(map[string]bool)
	
	for _, tool := range result.Tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("Expected tool %q not found", expected)
		}
	}

	if len(result.Tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(result.Tools))
	}
}

// TestEchoTool tests the echo tool implementation
func TestEchoTool(t *testing.T) {
	server := mcp.NewServer("test-server", "1.0.0")
	registerTools(server)

	// Get the call tool handler
	callHandler := server.GetCallToolHandler()
	if callHandler == nil {
		t.Fatal("No call tool handler found")
	}

	// Test echo with message parameter (this server uses "message", not "text")
	t.Run("EchoWithMessage", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Name:      "echo",
			Arguments: json.RawMessage(`{"message": "hello world"}`),
		}

		result, err := callHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Echo failed: %v", err)
		}

		// Check the result
		expectedText := "Echo: hello world"
		found := false

		if content, ok := result.Content.([]interface{}); ok {
			for _, item := range content {
				if m, ok := item.(map[string]interface{}); ok {
					if text, ok := m["text"].(string); ok && text == expectedText {
						found = true
						break
					}
				}
			}
		}

		if !found {
			t.Errorf("Expected text %q in response, got: %+v", expectedText, result.Content)
		}
	})

	// Test echo with missing message
	t.Run("EchoMissingMessage", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Name:      "echo",
			Arguments: json.RawMessage(`{}`),
		}

		_, err := callHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error for missing message parameter, got nil")
		}
	})
}

// TestRandomTool tests the random number generator tool
func TestRandomTool(t *testing.T) {
	server := mcp.NewServer("test-server", "1.0.0")
	registerTools(server)

	callHandler := server.GetCallToolHandler()
	if callHandler == nil {
		t.Fatal("No call tool handler found")
	}

	t.Run("RandomInRange", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Name:      "random",
			Arguments: json.RawMessage(`{"min": 10, "max": 20}`),
		}

		result, err := callHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Random failed: %v", err)
		}

		// Check that we got a result with a number in the expected range
		if content, ok := result.Content.([]interface{}); ok && len(content) > 0 {
			if item, ok := content[0].(map[string]interface{}); ok {
				if text, ok := item["text"].(string); ok {
					// The text should contain a number between 10 and 20
					var response struct {
						Number float64 `json:"number"`
					}
					if jsonText, ok := item["text"].(string); ok {
						if err := json.Unmarshal([]byte(jsonText), &response); err == nil {
							if response.Number < 10 || response.Number > 20 {
								t.Errorf("Random number %f is out of range [10, 20]", response.Number)
							}
						}
					}
				}
			}
		}
	})
}

// TestCurrentTimeTool tests the current time tool
func TestCurrentTimeTool(t *testing.T) {
	server := mcp.NewServer("test-server", "1.0.0")
	registerTools(server)

	callHandler := server.GetCallToolHandler()
	if callHandler == nil {
		t.Fatal("No call tool handler found")
	}

	t.Run("TimeWithTimezone", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Name:      "current_time",
			Arguments: json.RawMessage(`{"timezone": "UTC"}`),
		}

		result, err := callHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Current time failed: %v", err)
		}

		// Check that we got a result with time information
		if content, ok := result.Content.([]interface{}); ok && len(content) > 0 {
			if item, ok := content[0].(map[string]interface{}); ok {
				if _, hasText := item["text"]; !hasText {
					t.Error("Expected text in response")
				}
			}
		}
	})

	t.Run("TimeWithInvalidTimezone", func(t *testing.T) {
		req := mcp.CallToolRequest{
			Name:      "current_time",
			Arguments: json.RawMessage(`{"timezone": "Invalid/Zone"}`),
		}

		_, err := callHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid timezone, got nil")
		}
	})
}