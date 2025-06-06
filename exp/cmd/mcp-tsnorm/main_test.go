package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTSNormBasic(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcp-tsnorm-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test input file with an MCP trace header
	inputFile := filepath.Join(tempDir, "input.mcp")
	if err := os.WriteFile(inputFile, []byte("# mcptrace:v1\nmcp-recv {\"id\":1,\"method\":\"test\"} # 1234567890.123\nmcp-send {\"id\":1,\"result\":\"ok\"} # 1234567890.456\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Define output file
	outputFile := filepath.Join(tempDir, "output.mcp")

	// Run mcp-tsnorm with header preservation
	cmd := exec.Command("go", "run", "main.go", "-o", outputFile, inputFile)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("mcp-tsnorm failed: %v", err)
	}

	// Read the output file
	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Verify that the header was preserved
	outputStr := string(output)
	if !strings.Contains(outputStr, "# mcptrace:v1") {
		t.Error("header was not preserved")
	}

	// Verify that timestamps were normalized to start at 0
	if !strings.Contains(outputStr, "mcp-recv {\"id\":1,\"method\":\"test\"} # 0.000") {
		t.Error("first timestamp was not normalized to 0.000")
	}

	// Verify that relative timing was preserved
	if !strings.Contains(outputStr, "mcp-send {\"id\":1,\"result\":\"ok\"} # 0.333") {
		t.Error("relative timing was not preserved correctly")
	}
}

func TestAbsoluteTimestamps(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcp-tsnorm-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test input file with an MCP trace header
	inputFile := filepath.Join(tempDir, "input.mcp")
	if err := os.WriteFile(inputFile, []byte("# mcptrace:v1\nmcp-recv {\"id\":1,\"method\":\"test\"} # 1234567890.123\nmcp-send {\"id\":1,\"result\":\"ok\"} # 1234567890.456\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Define output file
	outputFile := filepath.Join(tempDir, "output.mcp")

	// Run mcp-tsnorm with absolute timestamps
	cmd := exec.Command("go", "run", "main.go", "-absolute", "1600000000", "-o", outputFile, inputFile)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("mcp-tsnorm failed: %v", err)
	}

	// Read the output file
	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Verify that timestamps were rebased to the absolute value
	outputStr := string(output)
	if !strings.Contains(outputStr, "mcp-recv {\"id\":1,\"method\":\"test\"} # 1600000000.000") {
		t.Error("timestamp was not rebased to absolute value")
	}
}

func TestHeaderStripping(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcp-tsnorm-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test input file with an MCP trace header
	inputFile := filepath.Join(tempDir, "input.mcp")
	if err := os.WriteFile(inputFile, []byte("# mcptrace:v1\nmcp-recv {\"id\":1,\"method\":\"test\"} # 1234567890.123\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Define output file
	outputFile := filepath.Join(tempDir, "output.mcp")

	// Run mcp-tsnorm with header stripping
	cmd := exec.Command("go", "run", "main.go", "-preserve-header=false", "-o", outputFile, inputFile)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("mcp-tsnorm failed: %v", err)
	}

	// Read the output file
	output, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	// Verify that the header was not included
	outputStr := string(output)
	if strings.Contains(outputStr, "# mcptrace:v1") {
		t.Error("header was not stripped")
	}
}
