// +build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/tmc/mcp"
)

func main() {
	server := mcp.NewServer("test-server", "1.0.0")
	
	// Add a simple tool for testing
	server.AddTool("echo", "Echo back the input", map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"message": map[string]interface{}{
				"type": "string",
				"description": "Message to echo",
			},
		},
		"required": []string{"message"},
	}, func(ctx context.Context, params map[string]interface{}) (*mcp.CallToolResult, error) {
		message, ok := params["message"].(string)
		if !ok {
			message = "No message provided"
		}
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": message,
				},
			},
		}, nil
	})
	
	// Run the server
	if err := mcp.ServeStdio(context.Background(), server); err != nil {
		log.Fatal(err)
	}
}