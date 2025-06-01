package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/testutil"
)

// Example demonstrates using the ServerClientPair helper for testing,
// similar to the golang-tools-internal-mcp pattern
func Example_serverClientPairHelper() {
	// Create a test server with capabilities
	server := mcp.NewServer("example-server", "1.0.0")

	// Register a tool
	tool := mcp.Tool{
		Name:        "greet",
		Description: "Greet someone",
		InputSchema: []byte(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"}
			},
			"required": ["name"]
		}`),
	}

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params struct {
			Name string `json:"name"`
		}

		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, err
		}

		greeting := fmt.Sprintf("Hello, %s! Welcome to MCP.", params.Name)

		return &mcp.CallToolResult{
			Content: []any{
				mcp.TextContent{
					Type: "text",
					Text: greeting,
				},
			},
		}, nil
	}

	if err := server.RegisterTool(tool, handler); err != nil {
		panic(err)
	}

	// Create a connected server/client pair
	// Note: examples don't have access to testing.T, so we'll skip the logger setup
	// In real tests, you would pass t as the first parameter
	ctx := context.Background()
	pair, err := testutil.NewServerClientPair(nil, ctx, server)
	if err != nil {
		panic(err)
	}
	defer pair.Cleanup()

	// Use the client to interact with the server
	result, err := pair.Client.CallTool(ctx, mcp.CallToolRequest{
		Name:      "greet",
		Arguments: []byte(`{"name": "Alice"}`),
	})
	if err != nil {
		panic(err)
	}

	// Extract the result (note: returns as map in practice)
	if len(result.Content) > 0 {
		if contentMap, ok := result.Content[0].(map[string]interface{}); ok {
			if text, ok := contentMap["text"].(string); ok {
				fmt.Println(text)
			}
		}
	}

	// Output: Hello, Alice! Welcome to MCP.
}

// Example demonstrates a more complex testing scenario with prompts and tools
func Example_advancedServerClientPair() {
	// Create a server configured for testing
	server := mcp.NewServer("test-server", "1.0.0")

	// Register both tools and prompts
	// ... tool registration ...

	// Create the test pair
	ctx := context.Background()
	pair, err := testutil.NewServerClientPair(nil, ctx, server)
	if err != nil {
		panic(err)
	}
	defer pair.Cleanup()

	// Test multiple operations
	_, err = pair.Client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		panic(err)
	}

	// The server and client are fully connected and ready for testing
	fmt.Println("Server and client connected successfully")

	// Output: Server and client connected successfully
}
