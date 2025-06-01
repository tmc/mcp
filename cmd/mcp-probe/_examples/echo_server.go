package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/internal/jsonrpc2util"
	"golang.org/x/exp/jsonrpc2"
)

func main() {
	// Create a simple echo server
	server := mcp.NewServer("echo-server", "1.0.0")

	// Register an echo tool
	inputSchema, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Message to echo",
			},
		},
		"required": []string{"message"},
	})

	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "Echo back the message",
		InputSchema: json.RawMessage(inputSchema),
	}

	server.RegisterTool(echoTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var params map[string]any
		if err := json.Unmarshal(req.Arguments, &params); err != nil {
			return nil, err
		}

		message, ok := params["message"].(string)
		if !ok {
			message = "no message provided"
		}

		return &mcp.CallToolResult{
			Content: []any{
				map[string]any{
					"type": "text",
					"text": message,
				},
			},
		}, nil
	})

	// Set up stdio transport
	rw := struct {
		io.Reader
		io.Writer
	}{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}

	// Set up connection with framing
	conn := jsonrpc2.NewConn(
		context.Background(),
		jsonrpc2util.NewStdioFramer(rw),
		jsonrpc2util.NewIdGeneratingBinder(server),
	)

	// Wait for completion
	<-conn.Done()
	if err := conn.Err(); err != nil {
		log.Fatal(err)
	}
}