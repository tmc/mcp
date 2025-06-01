package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestSessionPlayback(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-replay-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test script file
	scriptContent := `# MCP Session Recording
#
# Name: Test Session
# Date: 2023-01-01T00:00:00Z
#
# -> indicates a request or notification
# <- indicates a response or error
#
-> {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
<- {"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "test-server", "version": "1.0.0"}}}
-> {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
<- {"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "tool1", "description": "A test tool"}]}}
-> {"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": {}}
<- {"jsonrpc": "2.0", "id": 3, "result": {"success": true}}
`
	scriptPath := filepath.Join(tempDir, "session.mcp")
	err = os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("failed to write script file: %v", err)
	}

	// Test cases for session playback
	testCases := []struct {
		name           string
		args           []string
		wantErr        bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "basic playback",
			args: []string{"--script", scriptPath, "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				// Check for key methods in output
				if !strings.Contains(output, "initialize") {
					t.Error("output missing initialize method")
				}
				if !strings.Contains(output, "tools/list") {
					t.Error("output missing tools/list method")
				}
			},
		},
		{
			name: "verbose mode",
			args: []string{"--script", scriptPath, "--verbose", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				// Check for detailed output in verbose mode
				if !strings.Contains(output, "Session metadata:") {
					t.Error("output missing session metadata")
				}
				if !strings.Contains(output, "Command") {
					t.Error("output missing command information")
				}
			},
		},
		{
			name: "delay option",
			args: []string{"--script", scriptPath, "--delay", "0.1", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Delay: 0.1") {
					t.Error("output missing delay setting")
				}
			},
		},
		{
			name: "endpoint specification",
			args: []string{"--script", scriptPath, "--endpoint", "http://localhost:8080", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Endpoint: http://localhost:8080") {
					t.Error("output missing endpoint information")
				}
			},
		},
		{
			name: "output file",
			args: []string{"--script", scriptPath, "--output", filepath.Join(tempDir, "output.log"), "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Output: ") {
					t.Error("output missing output file information")
				}
			},
		},
		{
			name: "filter by method",
			args: []string{"--script", scriptPath, "--method", "initialize", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "initialize") {
					t.Error("output missing initialize method")
				}
				if strings.Contains(output, "tools/list") {
					t.Error("output should not include tools/list when filtering by initialize")
				}
			},
		},
		{
			name: "filter by range",
			args: []string{"--script", scriptPath, "--range", "1-2", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "initialize") {
					t.Error("output missing initialize method")
				}
				if !strings.Contains(output, "Playing commands 1-2") {
					t.Error("output missing range information")
				}
				if strings.Contains(output, "shutdown") {
					t.Error("output should not include shutdown when using range 1-2")
				}
			},
		},
		{
			name:    "invalid script file",
			args:    []string{"--script", "nonexistent.mcp"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewReplayCommand()
			stdout, stderr, err := test.RunCommand(t, cmd, tc.args)

			// Check error state
			if (err != nil) != tc.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v\nStderr: %s", err, tc.wantErr, stderr)
				return
			}

			// If we expected success, validate the output
			if !tc.wantErr && tc.validateOutput != nil {
				tc.validateOutput(t, stdout)
			}
		})
	}
}

func TestReplayValidation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-validation-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test script with expected responses
	validationScript := `# MCP Session with expected responses
-> {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
<- {"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "*"}}}
-> {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
<- {"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "*"}]}}
`
	scriptPath := filepath.Join(tempDir, "validation.mcp")
	err = os.WriteFile(scriptPath, []byte(validationScript), 0644)
	if err != nil {
		t.Fatalf("failed to write validation script file: %v", err)
	}

	// Create a sample response file to simulate server responses
	responseData := `{"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "test-server", "version": "1.0.0"}}}
{"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "tool1", "description": "A test tool"}]}}
`
	responsePath := filepath.Join(tempDir, "responses.json")
	err = os.WriteFile(responsePath, []byte(responseData), 0644)
	if err != nil {
		t.Fatalf("failed to write response file: %v", err)
	}

	// Test validation features
	testCases := []struct {
		name           string
		args           []string
		wantErr        bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "validate responses against script",
			args: []string{
				"--script", scriptPath,
				"--validate",
				"--response-file", responsePath,
				"--dry-run",
			},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Validation mode") {
					t.Error("output missing validation mode")
				}
				if !strings.Contains(output, "Response validation successful") {
					t.Error("output missing validation success message")
				}
			},
		},
		{
			name: "strict validation mode",
			args: []string{
				"--script", scriptPath,
				"--validate",
				"--strict",
				"--response-file", responsePath,
				"--dry-run",
			},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Strict validation") {
					t.Error("output missing strict validation mode")
				}
			},
		},
		{
			name: "validation with missing response",
			args: []string{
				"--script", scriptPath,
				"--validate",
				"--response-file", tempDir + "/nonexistent.json",
				"--dry-run",
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewReplayCommand()
			stdout, stderr, err := test.RunCommand(t, cmd, tc.args)

			// Check error state
			if (err != nil) != tc.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v\nStderr: %s", err, tc.wantErr, stderr)
				return
			}

			// If we expected success, validate the output
			if !tc.wantErr && tc.validateOutput != nil {
				tc.validateOutput(t, stdout)
			}
		})
	}
}
