// Package sdk2 provides a stdlib-idiomatic, well-typed Go API for the Model Context Protocol (MCP).
//
// This package prioritizes developer ergonomics and follows Go standard library patterns
// for a clean, type-safe interface to MCP functionality.
//
// Basic usage:
//
//	// Client
//	client := sdk2.NewClient(transport, sdk2.WithTimeout(30*time.Second))
//	defer client.Close()
//
//	tools, err := client.ListTools(ctx)
//	result, err := client.CallTool(ctx, "calculator", map[string]any{"op": "+", "a": 1, "b": 2})
//
//	// Server
//	server := sdk2.NewServer("my-server", "1.0.0")
//	server.AddTool("calculator", calculator.Tool())
//
//	if err := server.Serve(ctx, transport); err != nil {
//	    log.Fatal(err)
//	}
package sdk2
