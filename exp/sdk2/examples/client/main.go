// Package main demonstrates a simple client using SDK2.
package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/tmc/mcp/exp/sdk2"
)

func main() {
	// Start the calculator server as a subprocess
	cmd := exec.Command("go", "run", "../calculator/main.go")

	// Get stdin/stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to get stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to get stdout pipe: %v", err)
	}

	// Start the server process
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Create a transport that combines stdin/stdout
	transport := &pipeTransport{
		reader: stdout,
		writer: stdin,
		cmd:    cmd,
	}

	// Create client with options
	client := sdk2.NewClient(
		transport,
		sdk2.WithTimeout(10*time.Second),
		sdk2.WithRetries(3, time.Second),
	)
	defer client.Close()

	ctx := context.Background()

	// List available tools
	fmt.Println("Listing tools...")
	tools, err := client.ListTools(ctx)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	fmt.Printf("Found %d tools:\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Call the calculator tool
	fmt.Println("\nCalling calculator tool...")
	result, err := client.CallTool(ctx, "calculator", map[string]any{
		"operation": "add",
		"a":         5.0,
		"b":         3.0,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Printf("Calculator result: %+v\n", result)

	// Try division
	fmt.Println("\nTrying division...")
	result, err = client.CallTool(ctx, "calculator", map[string]any{
		"operation": "divide",
		"a":         10.0,
		"b":         2.0,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Printf("Division result: %+v\n", result)

	// Try division by zero
	fmt.Println("\nTrying division by zero...")
	result, err = client.CallTool(ctx, "calculator", map[string]any{
		"operation": "divide",
		"a":         10.0,
		"b":         0.0,
	})
	if err != nil {
		log.Fatalf("Failed to call tool: %v", err)
	}

	fmt.Printf("Division by zero result: %+v\n", result)
}

// pipeTransport implements Transport using command pipes.
type pipeTransport struct {
	reader interface{ Read([]byte) (int, error) }
	writer interface{ Write([]byte) (int, error) }
	cmd    *exec.Cmd
}

func (t *pipeTransport) Read(p []byte) (int, error) {
	return t.reader.Read(p)
}

func (t *pipeTransport) Write(p []byte) (int, error) {
	return t.writer.Write(p)
}

func (t *pipeTransport) Close() error {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Kill()
	}
	return nil
}
