package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBasic tests basic functionality of the jsonrpc2gostruct tool
func TestBasic(t *testing.T) {
	// Build the tool
	cmd := exec.Command("go", "build")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build tool: %v", err)
	}

	// Create a test directory
	tempDir := t.TempDir()

	// Create a simple test schema
	schemaJSON := `{
  "type": "object",
  "description": "A simple test schema",
  "properties": {
    "name": {
      "type": "string",
      "description": "The name of the item"
    },
    "count": {
      "type": "integer",
      "description": "The count of items"
    },
    "active": {
      "type": "boolean",
      "description": "Whether the item is active"
    }
  },
  "required": ["name", "active"]
}`
	schemaPath := filepath.Join(tempDir, "schema.json")
	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Create expected output
	expectedOutput := "Schema - A simple test schema"

	// Run the tool
	var stdout, stderr bytes.Buffer
	toolCmd := exec.Command("./jsonrpc2gostruct", "-package", "test", schemaPath)
	toolCmd.Stdout = &stdout
	toolCmd.Stderr = &stderr
	if err := toolCmd.Run(); err != nil {
		t.Fatalf("Tool execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// Check the output
	output := stdout.String()
	t.Logf("Tool output: %s", output)

	if !strings.Contains(output, "package test") {
		t.Errorf("Expected output to contain 'package test', got: %s", output)
	}

	if !strings.Contains(output, expectedOutput) {
		t.Errorf("Expected output to contain '%s', got: %s", expectedOutput, output)
	}

	if !strings.Contains(output, "Active bool `json:\"active\"`") {
		t.Errorf("Expected output to contain Active field, got: %s", output)
	}

	if !strings.Contains(output, "Name string `json:\"name\"`") {
		t.Errorf("Expected output to contain Name field, got: %s", output)
	}

	// Test with stdout
	outputPath := filepath.Join(tempDir, "output.go")
	outCmd := exec.Command("./jsonrpc2gostruct", "-package", "test", "-out", outputPath, schemaPath)
	if err := outCmd.Run(); err != nil {
		t.Fatalf("Tool execution with output file failed: %v", err)
	}

	// Check the file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created at: %s", outputPath)
	}

	// Read the output file
	outputContent, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Check the output file content
	if !strings.Contains(string(outputContent), "package test") {
		t.Errorf("Expected output file to contain 'package test'")
	}

	if !strings.Contains(string(outputContent), expectedOutput) {
		t.Errorf("Expected output file to contain '%s'", expectedOutput)
	}
}

// TestComplexSchema tests handling of complex schema with nested types and special formats
func TestComplexSchema(t *testing.T) {
	// Build the tool
	cmd := exec.Command("go", "build")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build tool: %v", err)
	}

	// Create a test directory
	tempDir := t.TempDir()

	// Create a complex test schema
	schemaJSON := `{
  "type": "object",
  "description": "A complex test schema",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique identifier"
    },
    "created": {
      "type": "string",
      "format": "date-time",
      "description": "Creation timestamp"
    },
    "tags": {
      "type": "array",
      "description": "List of tags",
      "items": {
        "type": "string"
      }
    },
    "metadata": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Additional metadata"
    }
  }
}`
	schemaPath := filepath.Join(tempDir, "complex.json")
	if err := os.WriteFile(schemaPath, []byte(schemaJSON), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Run the tool
	var stdout, stderr bytes.Buffer
	toolCmd := exec.Command("./jsonrpc2gostruct", "-package", "complex", schemaPath)
	toolCmd.Stdout = &stdout
	toolCmd.Stderr = &stderr
	if err := toolCmd.Run(); err != nil {
		t.Fatalf("Tool execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// Check the output
	output := stdout.String()
	t.Logf("Tool output: %s", output)

	if !strings.Contains(output, "Created time.Time") {
		t.Errorf("Expected output to use time.Time for date-time format, got: %s", output)
	}

	if !strings.Contains(output, "Tags []string") {
		t.Errorf("Expected output to use []string for array of strings, got: %s", output)
	}

	if !strings.Contains(output, `import (
	"encoding/json"
	"time"
)`) {
		t.Errorf("Expected output to import time package for date-time fields, got: %s", output)
	}
}

// TestJSONRPCMessage tests parsing of JSON-RPC messages
func TestJSONRPCMessage(t *testing.T) {
	// Build the tool
	cmd := exec.Command("go", "build")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build tool: %v", err)
	}

	// Create a test directory
	tempDir := t.TempDir()

	// Create a JSON-RPC request message
	jsonrpcJSON := `{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "calculator",
    "arguments": {
      "operation": "add",
      "a": 5,
      "b": 3
    }
  }
}`
	jsonrpcPath := filepath.Join(tempDir, "jsonrpc.json")
	if err := os.WriteFile(jsonrpcPath, []byte(jsonrpcJSON), 0644); err != nil {
		t.Fatalf("Failed to write JSON-RPC file: %v", err)
	}

	// Run the tool
	var stdout, stderr bytes.Buffer
	toolCmd := exec.Command("./jsonrpc2gostruct", "-package", "rpc", jsonrpcPath)
	toolCmd.Stdout = &stdout
	toolCmd.Stderr = &stderr
	if err := toolCmd.Run(); err != nil {
		t.Fatalf("Tool execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// Check the output
	output := stdout.String()
	t.Logf("Tool output: %s", output)

	// Verify JSON-RPC specific parts
	if !strings.Contains(output, "ToolsCallRequest") {
		t.Errorf("Expected output to contain ToolsCallRequest struct")
	}

	if !strings.Contains(output, "Name string") && !strings.Contains(output, "Arguments") {
		t.Errorf("Expected output to contain Name and Arguments fields")
	}
}

// TestToolsResponse tests parsing of tools response with input schemas
func TestToolsResponse(t *testing.T) {
	// Build the tool
	cmd := exec.Command("go", "build")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build tool: %v", err)
	}

	// Create a test directory
	tempDir := t.TempDir()

	// Create a tools response message
	toolsJSON := `{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "calculator",
        "description": "A simple calculator tool",
        "inputSchema": {
          "type": "object",
          "properties": {
            "operation": {
              "type": "string",
              "description": "The operation to perform"
            },
            "a": {
              "type": "number",
              "description": "First operand"
            },
            "b": {
              "type": "number",
              "description": "Second operand"
            }
          },
          "required": ["operation", "a", "b"]
        }
      }
    ]
  }
}`
	toolsPath := filepath.Join(tempDir, "tools.json")
	if err := os.WriteFile(toolsPath, []byte(toolsJSON), 0644); err != nil {
		t.Fatalf("Failed to write tools file: %v", err)
	}

	// Run the tool
	var stdout, stderr bytes.Buffer
	toolCmd := exec.Command("./jsonrpc2gostruct", "-package", "tools", toolsPath)
	toolCmd.Stdout = &stdout
	toolCmd.Stderr = &stderr
	if err := toolCmd.Run(); err != nil {
		t.Fatalf("Tool execution failed: %v\nStderr: %s", err, stderr.String())
	}

	// Check the output
	output := stdout.String()
	t.Logf("Tool output: %s", output)

	// Verify tools response specific parts
	if !strings.Contains(output, "CalculatorInput") {
		t.Errorf("Expected output to contain CalculatorInput struct")
	}

	if !strings.Contains(output, "Operation string") ||
		!strings.Contains(output, "A float64") ||
		!strings.Contains(output, "B float64") {
		t.Errorf("Expected output to contain calculator tool fields")
	}
}
