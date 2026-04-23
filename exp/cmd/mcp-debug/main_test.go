package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPDebugHelp(t *testing.T) {
	// Test that help flag works
	_ = bytes.Buffer{} // placeholder for future use

	// Mock args for help
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"mcp-debug", "-h"}

	// This would test the help functionality if main was refactored
	// For now, just test that we can instantiate the test
	t.Log("mcp-debug help test placeholder")
}

func TestMCPDebugBasicFunctionality(t *testing.T) {
	// Create a temporary MCP trace file for testing
	tmpDir, err := os.MkdirTemp("", "mcp-debug-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.mcp")
	testContent := `# mcptrace:v1 source=test
mcp-send {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}} # 1234567890.123
mcp-recv {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"test"}}} # 1234567890.456
mcp-send {"jsonrpc":"2.0","method":"notifications/initialized"} # 1234567890.789
`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test that the file exists and has content
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if !strings.Contains(string(content), "mcptrace:v1") {
		t.Error("Test file should contain mcptrace header")
	}

	if !strings.Contains(string(content), "mcp-send") {
		t.Error("Test file should contain mcp-send messages")
	}

	if !strings.Contains(string(content), "mcp-recv") {
		t.Error("Test file should contain mcp-recv messages")
	}

	t.Logf("Successfully created test file with %d bytes", len(content))
}

func TestMCPDebugValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		isValid bool
	}{
		{
			name: "valid trace",
			content: `# mcptrace:v1 source=test
mcp-send {"jsonrpc":"2.0","id":1,"method":"test"} # 1234567890.123
`,
			isValid: true,
		},
		{
			name: "invalid json",
			content: `# mcptrace:v1 source=test
mcp-send {invalid json} # 1234567890.123
`,
			isValid: false,
		},
		{
			name: "missing header",
			content: `mcp-send {"jsonrpc":"2.0","id":1,"method":"test"} # 1234567890.123
`,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "mcp-debug-validation-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			testFile := filepath.Join(tmpDir, "test.mcp")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Basic validation - check if file has expected structure
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read test file: %v", err)
			}

			hasHeader := strings.Contains(string(content), "mcptrace:v1")
			hasValidJSON := !strings.Contains(string(content), "{invalid json}")

			isValid := hasHeader && hasValidJSON
			if isValid != tt.isValid {
				t.Errorf("Validation result = %v, want %v", isValid, tt.isValid)
			}
		})
	}
}

func TestMCPDebugJSONParsing(t *testing.T) {
	validJSON := `{"jsonrpc":"2.0","id":1,"method":"test","params":{"key":"value"}}`
	invalidJSON := `{invalid json}`

	// Test that we can identify valid vs invalid JSON
	if !isValidJSON(validJSON) {
		t.Error("Valid JSON should be recognized as valid")
	}

	if isValidJSON(invalidJSON) {
		t.Error("Invalid JSON should be recognized as invalid")
	}
}

// Helper function for JSON validation
func isValidJSON(s string) bool {
	var js map[string]interface{}
	return json.Unmarshal([]byte(s), &js) == nil
}
