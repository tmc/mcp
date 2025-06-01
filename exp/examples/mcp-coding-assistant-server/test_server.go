package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"
)

// Test utility to test the server functionality without starting an actual server
func runToolTest(t *testing.T, toolName string, input interface{}) (map[string]interface{}, error) {
	// Convert input to JSON
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("error marshaling input: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process the tool call based on tool name
	var result interface{}
	switch toolName {
	case "Task":
		var taskInput TaskInput
		if err := json.Unmarshal(inputJSON, &taskInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Task input: %w", err)
		}
		result, err = TaskHandler(ctx, taskInput)
	case "Bash":
		var bashInput BashInput
		if err := json.Unmarshal(inputJSON, &bashInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Bash input: %w", err)
		}
		result, err = BashHandler(ctx, bashInput)
	case "Glob":
		var globInput GlobInput
		if err := json.Unmarshal(inputJSON, &globInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Glob input: %w", err)
		}
		result, err = GlobHandler(ctx, globInput)
	case "Grep":
		var grepInput GrepInput
		if err := json.Unmarshal(inputJSON, &grepInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Grep input: %w", err)
		}
		result, err = GrepHandler(ctx, grepInput)
	case "LS":
		var lsInput LSInput
		if err := json.Unmarshal(inputJSON, &lsInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling LS input: %w", err)
		}
		result, err = LSHandler(ctx, lsInput)
	case "Read":
		var readInput ReadInput
		if err := json.Unmarshal(inputJSON, &readInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Read input: %w", err)
		}
		result, err = ReadHandler(ctx, readInput)
	case "Edit":
		var editInput EditInput
		if err := json.Unmarshal(inputJSON, &editInput); err != nil {
			return nil, fmt.Errorf("error unmarshaling Edit input: %w", err)
		}
		result, err = EditHandler(ctx, editInput)
	default:
		return nil, fmt.Errorf("unsupported tool for test: %s", toolName)
	}

	if err != nil {
		return nil, fmt.Errorf("error processing %s: %w", toolName, err)
	}

	// Try to convert the result to the expected format
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	return resultMap, nil
}

// Test function to simulate an MCP client
func simulateMCPClient(t *testing.T) {
	// Create a fake MCP client
	client := struct {
		serverURL string
		jsonRPCID int
	}{
		serverURL: "http://localhost:8080",
		jsonRPCID: 1,
	}

	// Simulate a client request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      client.jsonRPCID,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "Bash",
			"arguments": map[string]interface{}{
				"command": "echo 'Hello, MCP!'",
			},
		},
	}

	// Print the request
	requestJSON, _ := json.MarshalIndent(request, "", "  ")
	log.Printf("Simulated client request:\n%s", string(requestJSON))

	// Run the tool directly
	result, err := runToolTest(t, "Bash", map[string]interface{}{
		"command": "echo 'Hello, MCP!'",
	})
	if err != nil {
		t.Fatalf("Error running tool: %v", err)
	}

	// Print the response
	responseJSON, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("Server response:\n%s", string(responseJSON))

	// Extract the actual stdout output
	content, ok := result["content"].([]map[string]interface{})
	if !ok || len(content) == 0 {
		t.Fatal("Invalid response format: missing content array")
	}

	textContent, ok := content[0]["text"].(string)
	if !ok {
		t.Fatal("Invalid response format: missing text in content")
	}

	var bashOutput map[string]interface{}
	if err := json.Unmarshal([]byte(textContent), &bashOutput); err != nil {
		t.Fatalf("Error unmarshaling bash output: %v", err)
	}

	stdout, ok := bashOutput["stdout"].(string)
	if !ok {
		t.Fatal("Invalid response format: missing stdout in bash output")
	}

	// Verify the output contains the expected string
	stdout = strings.TrimSpace(stdout)
	expected := "Hello, MCP!"
	if stdout != expected {
		t.Errorf("Expected output %q, got %q", expected, stdout)
	}

	log.Printf("Test passed: received expected output: %q", stdout)
}

// TestMCPServer runs a simulation of a client-server interaction
func TestMCPServer(t *testing.T) {
	// Set up the logging
	log.SetPrefix("[TEST] ")
	log.SetFlags(log.Ltime | log.Lshortfile)

	// Run the client simulation
	simulateMCPClient(t)
}

// Main function for running the test directly
func TestMain() {
	// Create a test object
	t := &testing.T{}
	TestMCPServer(t)
}
