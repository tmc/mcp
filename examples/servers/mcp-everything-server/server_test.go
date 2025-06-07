package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestServerBuild tests that the server builds successfully
func TestServerBuild(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "test-mcp-everything-server", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("test-mcp-everything-server")

	// Check that the binary exists
	if _, err := os.Stat("test-mcp-everything-server"); os.IsNotExist(err) {
		t.Fatal("Built binary does not exist")
	}
}

// TestServerProtocol tests basic MCP protocol interaction
func TestServerProtocol(t *testing.T) {
	// Build the server first
	cmd := exec.Command("go", "build", "-o", "test-mcp-everything-server", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("test-mcp-everything-server")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the server
	serverCmd := exec.CommandContext(ctx, "./test-mcp-everything-server", "-quiet")
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := serverCmd.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Send initialize request
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      0,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqJSON, err := json.Marshal(initReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	if _, err := fmt.Fprintln(stdin, string(reqJSON)); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	decoder := json.NewDecoder(stdout)
	var resp map[string]interface{}
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check response
	if resp["error"] != nil {
		t.Fatalf("Initialize failed: %v", resp["error"])
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if serverInfo, ok := result["serverInfo"].(map[string]interface{}); ok {
			if name, ok := serverInfo["name"].(string); ok {
				if name != "example-servers/everything" {
					t.Errorf("Expected server name 'example-servers/everything', got %s", name)
				}
			}
		}
	}

	// Test tools/list
	toolsReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	reqJSON, err = json.Marshal(toolsReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	if _, err := fmt.Fprintln(stdin, string(reqJSON)); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check that we got tools
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			expectedTools := []string{"echo", "current_time", "random"}
			foundTools := make(map[string]bool)

			for _, tool := range tools {
				if toolMap, ok := tool.(map[string]interface{}); ok {
					if name, ok := toolMap["name"].(string); ok {
						foundTools[name] = true
					}
				}
			}

			for _, expected := range expectedTools {
				if !foundTools[expected] {
					t.Errorf("Expected tool %s not found", expected)
				}
			}
		}
	}

	// Test echo tool
	echoReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "echo",
			"arguments": map[string]interface{}{
				"message": "hello test",
			},
		},
	}

	reqJSON, err = json.Marshal(echoReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	if _, err := fmt.Fprintln(stdin, string(reqJSON)); err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	// Read response
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check echo response
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if item, ok := content[0].(map[string]interface{}); ok {
				if text, ok := item["text"].(string); ok {
					expectedText := "Echo: hello test"
					if text != expectedText {
						t.Errorf("Expected echo text %q, got %q", expectedText, text)
					}
				}
			}
		}
	}

	// Send shutdown
	shutdownReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "shutdown",
	}

	reqJSON, err = json.Marshal(shutdownReq)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	if _, err := fmt.Fprintln(stdin, string(reqJSON)); err != nil && err != io.ErrClosedPipe {
		t.Logf("Write error (expected): %v", err)
	}

	// Close stdin
	stdin.Close()

	// Wait for server to exit
	serverCmd.Wait()
}

// TestServerFlags tests that server flags work correctly
func TestServerFlags(t *testing.T) {
	// Build the server
	cmd := exec.Command("go", "build", "-o", "test-mcp-everything-server", ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build server: %v", err)
	}
	defer os.Remove("test-mcp-everything-server")

	// Test with timeout flag
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	serverCmd := exec.CommandContext(ctx, "./test-mcp-everything-server", "-timeout", "100ms", "-quiet")
	startTime := time.Now()
	err := serverCmd.Run()
	duration := time.Since(startTime)

	// Check that the server exited within expected time
	if duration > 1*time.Second {
		t.Errorf("Server took too long to exit with timeout flag: %v", duration)
	}

	// The error might be nil if the server exited normally, or an exit error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// This is expected - server terminated after timeout
			_ = exitErr
		} else if !strings.Contains(err.Error(), "signal: killed") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}
