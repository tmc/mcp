package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/tmc/mcp"
	"github.com/tmc/mcp/modelcontextprotocol"
)

// TestEverythingServerIntegration tests the everything server integration
func TestEverythingServerIntegration(t *testing.T) {
	// Build the server
	cmd := exec.Command("go", "build", "-o", "mcp-everything-server", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("mcp-everything-server")

	// Test the server
	t.Run("BasicOperations", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "./mcp-everything-server")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("Failed to create stdin pipe: %v", err)
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatalf("Failed to create stdout pipe: %v", err)
		}

		if err := cmd.Start(); err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}

		// Test initialization
		initReq := modelcontextprotocol.Request{
			JSONRPC: "2.0",
			ID:      0,
			Method:  "initialize",
			Params: json.RawMessage(`{
				"protocolVersion": "2024-11-05",
				"clientInfo": {
					"name": "test-client",
					"version": "1.0.0"
				}
			}`),
		}

		reqJSON, err := json.Marshal(initReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		if _, err := stdin.Write(append(reqJSON, '\n')); err != nil {
			t.Fatalf("Failed to write request: %v", err)
		}

		// Read response
		decoder := json.NewDecoder(stdout)
		var resp modelcontextprotocol.Response
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("Initialization failed: %v", resp.Error)
		}

		// Test tools/list
		toolsReq := modelcontextprotocol.Request{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/list",
			Params:  json.RawMessage(`{}`),
		}

		reqJSON, err = json.Marshal(toolsReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		if _, err := stdin.Write(append(reqJSON, '\n')); err != nil {
			t.Fatalf("Failed to write request: %v", err)
		}

		// Read response
		if err := decoder.Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("Tools list failed: %v", resp.Error)
		}

		// Send shutdown
		shutdownReq := modelcontextprotocol.Request{
			JSONRPC: "2.0",
			Method:  "shutdown",
		}

		reqJSON, err = json.Marshal(shutdownReq)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		if _, err := stdin.Write(append(reqJSON, '\n')); err != nil && err != io.ErrClosedPipe {
			t.Logf("Write error (expected due to shutdown): %v", err)
		}

		// Close stdin to signal end of input
		stdin.Close()

		// Wait for server to exit
		cmd.Wait()
	})
}

// TestEchoToolImplementation specifically tests the echo tool implementation
func TestEchoToolImplementation(t *testing.T) {
	// Create a real server instance to test the echo tool directly
	server := setupTestServer()

	// Test echo with "message" parameter
	t.Run("EchoWithMessage", func(t *testing.T) {
		echoReq := modelcontextprotocol.CallToolRequest{
			Name:      "echo",
			Arguments: json.RawMessage(`{"message": "hello world"}`),
		}

		// Call the tool handler directly
		for _, tool := range server.tools {
			if tool.Name == "echo" {
				result, err := server.callToolHandler(context.Background(), echoReq)
				if err != nil {
					t.Fatalf("Echo tool failed: %v", err)
				}

				// Check the result
				expectedContent := "Echo: hello world"
				found := false
				if content, ok := result.Content.([]interface{}); ok {
					for _, item := range content {
						if m, ok := item.(map[string]interface{}); ok {
							if text, ok := m["text"].(string); ok && text == expectedContent {
								found = true
								break
							}
						}
					}
				}

				if !found {
					t.Errorf("Expected echo response %q, got: %+v", expectedContent, result.Content)
				}
				break
			}
		}
	})
}

// setupTestServer creates a test instance of the everything server
func setupTestServer() *mcp.Server {
	server := mcp.NewServer(ServerName, ServerVersion,
		mcp.WithServerInstructions("A comprehensive example MCP server with various capabilities"),
	)
	registerTools(server)
	return server
}
