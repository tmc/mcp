package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

func main() {
	// Create server with name and version
	srv := mcp.NewServer("echo-server", "1.0.0")

	// Register echo tool
	srv.RegisterTool("echo", "Echoes back the provided message along with a timestamp. Use this tool to get a response that includes your message and the current server time.", func(ctx context.Context, args map[string]json.RawMessage) (*modelcontextprotocol.CallToolResult, error) {
		// Parse message argument
		var messageRaw json.RawMessage
		var exists bool
		if messageRaw, exists = args["message"]; !exists {
			return nil, errors.New("missing required argument: message")
		}

		var message string
		if err := json.Unmarshal(messageRaw, &message); err != nil {
			return nil, fmt.Errorf("invalid message argument: %w", err)
		}

		if strings.TrimSpace(message) == "" {
			return nil, errors.New("message cannot be empty")
		}

		// Create response data
		responseData := map[string]interface{}{
			"echo":      message,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Log the echo to stderr
		log.Printf("[%s] Echo: \"%s\"", responseData["timestamp"], message)

		// Convert to JSON string
		responseJSON, err := json.Marshal(responseData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response: %w", err)
		}

		return &modelcontextprotocol.CallToolResult{
			Content: []modelcontextprotocol.Content{
				modelcontextprotocol.TextContent{
					Type: "text",
					Text: string(responseJSON),
				},
			},
		}, nil
	})

	// Start server with stdio transport
	transport := mcp.StdioTransport{}
	log.Println("Echo server running on stdio")

	if err := srv.Serve(context.Background(), transport); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}