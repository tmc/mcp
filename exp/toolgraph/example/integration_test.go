package example_test

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/tmc/mcp/client"
	"github.com/tmc/mcp/server"
)

func TestMCPIntegration(t *testing.T) {
	// Start MCP server
	srv := server.New()
	go srv.Start(":8080")
	defer srv.Stop()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test server initialization
	t.Run("Initialize", func(t *testing.T) {
		client := client.New("localhost:8080")
		err := client.Initialize("test-client")
		if err != nil {
			t.Fatalf("Failed to initialize: %v", err)
		}
	})

	// Test tool listing
	t.Run("ListTools", func(t *testing.T) {
		cmd := exec.Command("mcp-client", "list-tools")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to list tools: %v", err)
		}
		
		if !contains(string(output), "authenticate") {
			t.Error("Expected authenticate tool")
		}
	})

	// Test authentication
	t.Run("Authenticate", func(t *testing.T) {
		cmd := exec.Command("mcp-tool", "call", "authenticate", 
			`{"username": "test", "password": "pass"}`)
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Authentication failed: %v", err)
		}
		
		if !contains(string(output), "access_token") {
			t.Error("Expected access token in response")
		}
	})

	// Test file operations
	t.Run("FileOperations", func(t *testing.T) {
		// Create test file
		cmd := exec.Command("touch", "test.txt")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		
		// Write to file
		cmd = exec.Command("echo", "test data")
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
		
		// Read from file
		cmd = exec.Command("cat", "test.txt")
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}
		
		if string(output) != "test data\n" {
			t.Errorf("Unexpected file content: %s", output)
		}
		
		// Cleanup
		cmd = exec.Command("rm", "test.txt")
		cmd.Run()
	})
}

func TestToolExecution(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		args    string
		expect  string
	}{
		{
			name:   "Calculator",
			tool:   "calc",
			args:   `{"operation": "add", "x": 1, "y": 2}`,
			expect: `"result": 3`,
		},
		{
			name:   "Echo",
			tool:   "echo",
			args:   `{"message": "hello"}`,
			expect: `"response": "hello"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("mcp-tool", "call", tt.tool, tt.args)
			output, err := cmd.Output()
			if err != nil {
				t.Fatalf("Tool execution failed: %v", err)
			}
			
			if !contains(string(output), tt.expect) {
				t.Errorf("Expected %s in output, got: %s", tt.expect, output)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}