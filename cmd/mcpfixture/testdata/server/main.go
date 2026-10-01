package main

import (
	"context"
	"os"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := sdk.NewServer(&sdk.Implementation{Name: "fixture", Version: "1"}, nil)
	server.AddTool(&sdk.Tool{Name: "echo", InputSchema: map[string]any{"type": "object"}}, func(context.Context, *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		text := os.Getenv("MCP_RESPONSE")
		if text == "" {
			text = "hello"
		}
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: text}}}, nil
	})
	if server.Run(context.Background(), &sdk.StdioTransport{}) != nil {
		os.Exit(1)
	}
}
