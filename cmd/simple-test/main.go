package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/tmc/mcp"
)

func main() {
	// Start time server
	cmd := exec.Command("go", "run", "./examples/servers/mcp-time-server")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Give server time to start
	time.Sleep(2 * time.Second)

	// Create stdio transport
	transport := mcp.StdioTransport()

	// Create client
	client, err := mcp.NewClient(transport)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Initialize connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.Initialize(ctx, mcp.InitializeRequest{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    "test-client",
			Version: "1.0",
		},
	})

	if err != nil {
		log.Fatalf("Initialize failed: %v", err)
	}

	fmt.Printf("Server: %s %s\n", result.ServerInfo.Name, result.ServerInfo.Version)
}
