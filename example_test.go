package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/tmc/mcp"
)

// Example demonstrates the basic lifecycle of the MCP API: stand up a server
// with a tool, connect a client over an in-memory pipe, initialize, and call
// the tool.
func Example() {
	ctx := context.Background()

	// Wire a client and server together over an in-memory pipe.
	clientConn, serverConn := net.Pipe()
	clientTransport := &mcp.ReadWriteCloserTransport{ReadWriteCloser: clientConn}
	serverTransport := &mcp.ReadWriteCloserTransport{ReadWriteCloser: serverConn}

	server := mcp.NewServer("example-server", "1.0.0",
		mcp.WithServerInstructions("An example server for demonstrating the MCP SDK"))
	server.RegisterTool(mcp.Tool{
		Name:        "add",
		Description: "Add two numbers",
	}, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args struct {
			A int `json:"a"`
			B int `json:"b"`
		}
		if err := json.Unmarshal(req.Arguments, &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
		return &mcp.CallToolResult{
			Content: []any{mcp.TextContent{Type: "text", Text: fmt.Sprintf("%d", args.A+args.B)}},
		}, nil
	})

	go func() {
		// Serve returns an error when the connection closes, which is expected.
		_ = server.Serve(ctx, serverTransport)
	}()

	client, err := mcp.NewClient(clientTransport)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	init, err := client.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo:      mcp.Implementation{Name: "example-client", Version: "1.0.0"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Connected to %s (version %s)\n", init.ServerInfo.Name, init.ServerInfo.Version)
	fmt.Printf("Server instructions: %s\n", init.Instructions)

	tools, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Available tools:")
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	args, _ := json.Marshal(map[string]int{"a": 7, "b": 5})
	res, err := client.CallTool(ctx, mcp.CallToolRequest{Name: "add", Arguments: args})
	if err != nil {
		log.Fatal(err)
	}
	if content, ok := res.Content[0].(map[string]any); ok {
		fmt.Printf("Calculation result: %v\n", content["text"])
	}

	// Output:
	// Connected to example-server (version 1.0.0)
	// Server instructions: An example server for demonstrating the MCP SDK
	// Available tools:
	// - add: Add two numbers
	// Calculation result: 12
}
