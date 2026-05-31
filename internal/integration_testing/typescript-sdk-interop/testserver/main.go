package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tmc/mcp"
)

func main() {
	server := mcp.NewServer("tmc-mcp-typescript-stdio-smoke", "0.0.1")
	if err := server.RegisterTool(echoTool(), echo); err != nil {
		fmt.Fprintf(os.Stderr, "register echo tool: %v\n", err)
		os.Exit(1)
	}
	if err := server.Serve(context.Background(), mcp.StdioTransport()); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

func echoTool() mcp.Tool {
	return mcp.Tool{
		Name:        "echo",
		Description: "Echo a message for TypeScript SDK interop smoke tests.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string"
				}
			},
			"required": ["message"]
		}`),
	}
}

func echo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(req.Arguments, &args); err != nil {
		return nil, fmt.Errorf("decode echo arguments: %w", err)
	}
	if args.Message == "" {
		return nil, fmt.Errorf("message is required")
	}
	return &mcp.CallToolResult{
		Content: []any{
			mcp.TextContent{
				Type: "text",
				Text: "echo: " + args.Message,
			},
		},
	}, nil
}
