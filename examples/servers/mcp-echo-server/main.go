package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "echo-server"
	ServerVersion = "1.0.0"
)

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Echo Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("An echo server that returns messages with timestamps"),
	)

	// Register echo tool
	registerEchoTool(server)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerEchoTool(server *mcp.Server) {
	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "Echoes back the provided message along with a timestamp. Use this tool to get a response that includes your message and the current server time.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {
					"type": "string",
					"description": "The message to echo back"
				}
			},
			"required": ["message"]
		}`),
	}

	server.RegisterTool(echoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		message, ok := params["message"].(string)
		if !ok || message == "" {
			return nil, fmt.Errorf("message is required and must be a string")
		}

		if strings.TrimSpace(message) == "" {
			return nil, fmt.Errorf("message cannot be empty")
		}

		// Create response data
		responseData := map[string]interface{}{
			"echo":      message,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Log the echo to stderr
		log.Printf("[%s] Echo: \"%s\"", responseData["timestamp"], message)

		// Convert to JSON string
		responseJSON, _ := json.MarshalIndent(responseData, "", "  ")
		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": string(responseJSON),
				},
			},
		}, nil
	})

	log.Println("Registered echo tool")
}