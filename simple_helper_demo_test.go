package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/testutil"
)

// TestHelperDemo demonstrates the helper usage
func TestHelperDemo(t *testing.T) {
	// Create a server with a simple tool
	server := mcp.NewServer("demo-server", "1.0.0")

	// Register a simple tool
	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "Echo input back",
		InputSchema: []byte(`{"type": "object", "properties": {"message": {"type": "string"}}}`),
	}

	echoHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return nil, err
		}

		message, _ := args["message"].(string)

		return &mcp.CallToolResult{
			Content: []any{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Echo: %s", message),
				},
			},
		}, nil
	}

	if err := server.RegisterTool(echoTool, echoHandler); err != nil {
		t.Fatalf("Failed to register echo tool: %v", err)
	}

	// Create the server/client pair
	ctx := context.Background()
	pair, err := testutil.NewServerClientPair(t, ctx, server)
	if err != nil {
		t.Fatalf("Failed to create server/client pair: %v", err)
	}
	defer pair.Cleanup()

	// Test listing tools
	toolsResult, err := pair.Client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	// Test calling the tool
	echoResult, err := pair.Client.CallTool(ctx, mcp.CallToolRequest{
		Name:      "echo",
		Arguments: []byte(`{"message": "Hello, MCP!"}`),
	})
	if err != nil {
		t.Fatalf("Failed to call echo tool: %v", err)
	}

	if len(echoResult.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(echoResult.Content))
	}

	// The content comes back as a map, we need to check the structure
	if contentMap, ok := echoResult.Content[0].(map[string]interface{}); ok {
		if contentMap["type"] == "text" {
			expectedText := "Echo: Hello, MCP!"
			if text, ok := contentMap["text"].(string); ok {
				if text != expectedText {
					t.Errorf("Expected '%s', got '%s'", expectedText, text)
				}
			} else {
				t.Errorf("Expected text field to be string, got %T", contentMap["text"])
			}
		} else {
			t.Errorf("Expected type 'text', got '%s'", contentMap["type"])
		}
	} else {
		t.Errorf("Expected map[string]interface{}, got %T", echoResult.Content[0])
	}

	t.Log("Server/client pair helper test completed successfully!")
}
