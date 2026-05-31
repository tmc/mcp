/*
Package mcp provides Go implementations for the Model Context Protocol (MCP)
client and server. It defines core MCP types, interfaces for interaction, and
abstractions for transport mechanisms, aiming to provide an idiomatic Go API
for building MCP integrations.

This package relies on the standard Go library for transports and keeps its
JSON-RPC connection machinery internal.

# Overview

The Model Context Protocol (MCP) enables seamless integration between LLM applications
and external tools, resources, and prompts. This implementation provides:

- Complete support for the MCP specification
- Type-safe generic API with Go generics
- Tools, resources, and prompts management
- Client and server implementations
- Multiple transport options

# Client Example

Here's a basic example of using the client:

	// Create a transport
	transport := mcp.StdioTransport()

	// Create a client
	client, err := mcp.NewClient(transport)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Initialize the client
	ctx := context.Background()
	result, err := client.Initialize(ctx, mcp.InitializeRequest{
		ClientInfo: mcp.Implementation{
			Name:    "example-client",
			Version: "1.0.0",
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Printf("Connected to %s (version %s)\n",
		result.ServerInfo.Name, result.ServerInfo.Version)

	// List available tools
	toolsResult, err := client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Available tools:\n")
	for _, tool := range toolsResult.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

# Server Example

Here's a basic example of implementing a server:

	// Create a server
	server := mcp.NewServer("example-server", "1.0.0",
		mcp.WithServerInstructions("An example MCP server"),
	)

	// Register a type-safe tool
	type CalculatorInput struct {
		A int `json:"a"`
		B int `json:"b"`
	}

	type CalculatorOutput struct {
		Result int `json:"result"`
	}

	err := mcp.RegisterTypedToolWithServer(server, "add", "Add two numbers",
		func(ctx context.Context, input CalculatorInput) (CalculatorOutput, error) {
			return CalculatorOutput{Result: input.A + input.B}, nil
		})
	if err != nil {
		log.Fatalf("Failed to register tool: %v", err)
	}

	// Start the server (nil defaults to using stdin/stdout)
	if err := server.Serve(context.Background(), nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	// Or provide a custom transport function
	// customTransport := func(ctx context.Context) (io.ReadWriteCloser, error) {
	//     // Create or get your io.ReadWriteCloser here
	//     return myReadWriteCloser, nil
	// }
	// if err := server.Serve(context.Background(), customTransport); err != nil {
	//     log.Fatalf("Server error: %v", err)
	// }

# Notifications

The MCP protocol supports asynchronous notifications from server to client.
These can be handled using the OnNotification method on the client:

	client.OnNotification(func(notification mcp.JSONRPCNotification) {
		fmt.Printf("Received notification: %s\n", notification.Method)
		// Handle specific notification types
		switch notification.Method {
		case "tools/list/changed":
			// Refresh tool list
		case "progress":
			// Update progress UI
		}
	})

# Context Cancellation

The client automatically handles context cancellation by sending
notifications/cancelled messages to the server. When using context.WithCancelCause,
the cancellation reason is automatically propagated:

	ctx, cancel := context.WithCancelCause(context.Background())

	go func() {
		result, err := client.CallTool(ctx, mcp.CallToolRequest{
			Name: "analyze_data",
		})
		// Handle result or error
	}()

	// Cancel with a specific reason
	cancel(errors.New("user clicked stop button"))

	// The client automatically sends a notifications/cancelled message
	// with the reason to the server

# Working with Resources

Resources represent data that can be read from the server:

	// Register a resource on the server
	server.RegisterResource(mcp.Resource{
		URI:         "/data/config.json",
		Description: "Configuration",
		MimeType:    "application/json",
	}, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		// Return the resource content
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.URI,
				MimeType: "application/json",
				Text:     `{"debug":true}`,
			},
		}, nil
	})

	// Read a resource from the client
	result, err := client.ReadResource(ctx, mcp.ReadResourceRequest{
		URI: "/data/config.json",
	})
*/
package mcp
