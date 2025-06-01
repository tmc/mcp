package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/tmc/mcp"
)

const (
	ServerName    = "mcp-say-server"
	ServerVersion = "0.1.0"
)

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	log.Println("Starting MCP Say Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A simple MCP server that invokes macOS text-to-speech"),
	)

	// Register the say tool
	registerTools(server)

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, nil); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}

func registerTools(server *mcp.Server) {
	// Register say tool
	sayTool := mcp.Tool{
		Name:        "say",
		Description: "Speak text using macOS text-to-speech",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "The text to speak aloud"
				},
				"voice": {
					"type": "string",
					"description": "The voice to use (optional)"
				},
				"rate": {
					"type": "integer",
					"description": "Speech rate in words per minute (optional, default: 180)"
				}
			},
			"required": ["text"]
		}`),
	}

	server.RegisterTool(sayTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]interface{}
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, fmt.Errorf("invalid parameters: %v", err)
		}

		text, ok := params["text"].(string)
		if !ok || text == "" {
			return nil, fmt.Errorf("text is required and must be a string")
		}

		// Build the say command
		cmdArgs := []string{text}

		// Add voice if specified
		if voice, ok := params["voice"].(string); ok && voice != "" {
			cmdArgs = append([]string{"-v", voice}, cmdArgs...)
		}

		// Add rate if specified
		if rate, ok := params["rate"].(float64); ok {
			cmdArgs = append([]string{"-r", fmt.Sprintf("%.0f", rate)}, cmdArgs...)
		}

		// Execute the say command
		cmd := exec.CommandContext(ctx, "say", cmdArgs...)
		if err := cmd.Run(); err != nil {
			return &mcp.CallToolResult{
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": fmt.Sprintf("Failed to speak: %v", err),
					},
				},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResult{
			Content: []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": fmt.Sprintf("Successfully spoke: %q", text),
				},
			},
		}, nil
	})

	log.Println("Registered say tool")
}