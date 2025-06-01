package main

import (
	"context"
	"log"

	"github.com/tmc/mcprepos/mcp/adapters"
	"github.com/tmc/mcprepos/mcp/adapters/golang_tools"
	"github.com/tmc/mcprepos/mcp/adapters/mark3labs"
	"github.com/tmc/mcprepos/mcp/server"
	"github.com/tmc/mcprepos/mcp/transport"
)

func main() {
	// Example: Using mark3labs adapter
	usingMark3LabsAdapter()

	// Example: Using golang-tools adapter
	usingGolangToolsAdapter()
}

func usingMark3LabsAdapter() {
	// Create mark3labs adapter
	adapter := mark3labs.NewAdapter()

	// Register mark3labs tools (this would typically be done by importing mark3labs server code)
	// For example:
	// tool := mark3ServerInstance.GetTool("echo")
	// adapter.RegisterTool("echo", tool.Handler)

	// Create SDK server with adapter
	srv := server.NewServer("mark3labs-wrapped", "1.0.0", nil)
	srv.SetAdapter(adapter)

	// Create transport and serve
	transport := transport.NewStdIOTransport()
	ctx := context.Background()
	if err := srv.ServeTransport(ctx, transport); err != nil {
		log.Fatal(err)
	}
}

func usingGolangToolsAdapter() {
	// Create golang-tools adapter  
	adapter := golang_tools.NewAdapter()

	// Import golang-tools server (this would typically be done by importing golang-tools server code)
	// For example:
	// golangToolsServer := golangtools.NewServer("example", "1.0.0", nil)
	// adapter.SetServer(golangToolsServer)

	// Create SDK server with adapter
	srv := server.NewServer("golang-tools-wrapped", "1.0.0", nil)
	srv.SetAdapter(adapter)

	// Create transport and serve
	transport := transport.NewStdIOTransport()
	ctx := context.Background()
	if err := srv.ServeTransport(ctx, transport); err != nil {
		log.Fatal(err)
	}
}

// Example of adapter auto-detection
func autoDetectAdapter() {
	// The adapter registry can determine which adapter to use based on the server's type
	serverName := "my-server"

	// GetAdapter will return the appropriate adapter based on detection logic
	adapter := adapters.GetAdapter(serverName)
	if adapter == nil {
		log.Fatal("Could not determine appropriate adapter")
	}

	// Use the detected adapter
	srv := server.NewServer(serverName, "1.0.0", nil)
	srv.SetAdapter(adapter)

	// Continue with transport setup...
}