package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/tmc/mcp"
)

func Example() {
	// Create a server
	svc := mcp.NewService("example", "1.0.0")

	// Register a tool
	err := svc.RegisterTool(mcp.Tool{
		Name:        "echo",
		Description: "Echo the input",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
			},
		},
		Handler: func(ctx context.Context, args json.RawMessage) (*mcp.ToolResult, error) {
			var params struct {
				Message string `json:"message"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return nil, err
			}
			return &mcp.ToolResult{
				Content: []mcp.Content{{
					Type: "text",
					Text: params.Message,
				}},
			}, nil
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create server
	server := mcp.NewServer(svc)

	// Set up connection (example uses pipe)
	clientConn, serverConn := net.Pipe()

	// Serve in background
	go server.ServeConn(serverConn)

	// Create client
	c := mcp.NewClient(clientConn)
	defer c.Close()

	// Initialize
	reply, err := c.Initialize(context.Background(), mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	})
	fmt.Println(reply.Instructions)
	if err != nil {
		log.Fatal(err)
	}

	// Call tool
	result, err := c.CallTool(context.Background(), "echo", map[string]string{
		"message": "Hello, World!",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(result.Content[0].Text)
	// Output: Hello, World!
}
