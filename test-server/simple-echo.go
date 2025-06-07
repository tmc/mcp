package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tmc/mcp"
)

// StdioTransport provides a transport using stdin/stdout
type StdioTransport struct{}

// Dial implements the Transport interface
func (t *StdioTransport) Dial(ctx context.Context) (io.ReadWriteCloser, error) {
	return &stdioConn{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}, nil
}

// stdioConn wraps stdin/stdout as a ReadWriteCloser
type stdioConn struct {
	io.Reader
	io.Writer
}

// Close implements io.Closer
func (c *stdioConn) Close() error {
	// Don't actually close stdin/stdout
	return nil
}

func main() {
	// Redirect logs to stderr to keep stdout clean for the protocol
	log.SetOutput(os.Stderr)
	log.Println("Starting Simple Echo Server...")

	// Create a context that can be canceled
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a server
	server := mcp.NewServer("simple-echo", "1.0.0",
		mcp.WithServerInstructions("A simple echo server"),
	)

	// Register echo tool
	echoTool := mcp.Tool{
		Name:        "echo",
		Description: "Echo back a message",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Message to echo back",
				},
			},
			"required": []string{"message"},
		},
	}

	server.RegisterTool(echoTool, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		message, ok := args["message"].(string)
		if !ok {
			return nil, fmt.Errorf("message must be a string")
		}

		log.Printf("Echoing message: %s", message)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: fmt.Sprintf("Echo: %s", message),
				},
			},
		}, nil
	})

	log.Println("Tool registered successfully")

	// Create stdio transport
	stdioTransport := &StdioTransport{}

	// Serve via stdio
	log.Println("Starting protocol server via stdio...")
	if err := server.Serve(ctx, stdioTransport); err != nil {
		if err != context.Canceled {
			log.Fatalf("Error serving: %v", err)
		}
		log.Println("Server terminated.")
	}
}
