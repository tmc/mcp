package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestServerImplementations tests different MCP server implementations
func TestServerImplementations(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping server implementation tests in short mode")
	}

	// Define test cases for each server
	testCases := []struct {
		name       string
		checkFunc  func(t *testing.T) bool
		serverFunc func(t *testing.T, input string) (string, error)
	}{
		{
			name: "LocalMCPEverything",
			checkFunc: func(t *testing.T) bool {
				// Check if local server is available
				serverPath := "./mcp-everything-server"
				_, err := os.Stat(serverPath)
				if os.IsNotExist(err) {
					// Try to find it in PATH
					_, err = exec.LookPath("mcp-everything-server")
				}
				return err == nil
			},
			serverFunc: func(t *testing.T, input string) (string, error) {
				return runServer(t, "./mcp-everything-server", nil, input)
			},
		},
		{
			name: "Mark3LabsMCPEverything",
			checkFunc: func(t *testing.T) bool {
				// Check if Mark3Labs server is available
				serverPath := "/Users/tmc/go/src/github.com/tmc/mcprepos/mark3labs-mcp-go/examples/everything/mark3-everything-server"
				_, err := os.Stat(serverPath)
				return err == nil
			},
			serverFunc: func(t *testing.T, input string) (string, error) {
				return runServer(t, "/Users/tmc/go/src/github.com/tmc/mcprepos/mark3labs-mcp-go/examples/everything/mark3-everything-server", nil, input)
			},
		},
		{
			name: "NPXMCPEverything",
			checkFunc: func(t *testing.T) bool {
				// Check if NPX is available
				_, err := exec.LookPath("npx")
				return err == nil
			},
			serverFunc: func(t *testing.T, input string) (string, error) {
				return runServer(t, "npx", []string{"@modelcontextprotocol/server-everything", "stdio"}, input)
			},
		},
	}

	// Run tests for each server implementation
	for _, tc := range testCases {
		tc := tc // Capture for closure
		t.Run(tc.name, func(t *testing.T) {
			// Skip if server is not available
			if !tc.checkFunc(t) {
				t.Skipf("%s server not available, skipping", tc.name)
			}

			// Run basic protocol tests
			testServerBasicProtocol(t, tc.serverFunc)
		})
	}
}

// testServerBasicProtocol tests basic MCP protocol operations
func testServerBasicProtocol(t *testing.T, serverFunc func(t *testing.T, input string) (string, error)) {
	tests := []struct {
		name     string
		input    string
		contains []string // List of strings that should be in the output
	}{
		{
			name:     "Initialize",
			input:    `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0.0"}}}`,
			contains: []string{`"jsonrpc":"2.0"`, `"id":0`, `"protocolVersion"`},
		},
		{
			name:     "ToolsList",
			input:    `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
			contains: []string{`"jsonrpc":"2.0"`, `"id":1`, `"tools"`},
		},
		{
			name:     "EchoTool",
			input:    `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"echo","arguments":{"message":"test message"}}}`,
			contains: []string{`"jsonrpc":"2.0"`, `"id":2`, `"message":"test message"`},
		},
		{
			name:  "Exit",
			input: `{"jsonrpc":"2.0","method":"exit"}`,
			// No specific content to check for exit notification
		},
	}

	for _, tt := range tests {
		tt := tt // Capture for closure
		t.Run(tt.name, func(t *testing.T) {
			// Run the server with the test input
			output, err := serverFunc(t, tt.input)
			if err != nil {
				t.Fatalf("Failed to run server: %v", err)
			}

			// Check for expected content
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but got:\n%s", expected, output)
				}
			}
		})
	}
}

// runServer runs an MCP server with the given command and input
func runServer(t *testing.T, cmdPath string, args []string, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdPath, args...)

	// Set up input and output pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %v", err)
	}

	// Write input to the command
	if _, err := io.WriteString(stdin, input); err != nil {
		return "", fmt.Errorf("failed to write to stdin: %v", err)
	}
	stdin.Close()

	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") {
		// Return error if it's not just a timeout
		return "", fmt.Errorf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	// Return the combined output
	if stderr.Len() > 0 {
		return fmt.Sprintf("%s\nSTDERR: %s", stdout.String(), stderr.String()), nil
	}
	return stdout.String(), nil
}
