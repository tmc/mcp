package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestComplianceTests(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-test-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test server configuration
	serverConfigJSON := `{
		"name": "test-server",
		"version": "1.0.0",
		"transport": "stdio",
		"command": "cat",
		"tools": [
			{
				"name": "tool1",
				"description": "A test tool",
				"schema": {
					"type": "object",
					"properties": {
						"foo": {"type": "string"}
					},
					"required": ["foo"]
				}
			}
		]
	}`

	serverConfigPath := filepath.Join(tempDir, "server.json")
	err = os.WriteFile(serverConfigPath, []byte(serverConfigJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write server config file: %v", err)
	}

	// Test cases for different compliance tests
	testCases := []struct {
		name           string
		args           []string
		wantErr        bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "basic compliance test",
			args: []string{"--config", serverConfigPath, "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Compliance test plan:") {
					t.Error("output missing compliance test plan")
				}
			},
		},
		{
			name: "list tests only",
			args: []string{"--config", serverConfigPath, "--list-tests"},
			validateOutput: func(t *testing.T, output string) {
				expectedTests := []string{
					"initialize", "shutdown", "tools/list", "tools/call",
				}
				for _, test := range expectedTests {
					if !strings.Contains(output, test) {
						t.Errorf("output missing test %q", test)
					}
				}
			},
		},
		{
			name: "specific test",
			args: []string{"--config", serverConfigPath, "--test", "initialize", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Running test: initialize") {
					t.Error("output missing initialize test")
				}
			},
		},
		{
			name:    "invalid test",
			args:    []string{"--config", serverConfigPath, "--test", "nonexistent", "--dry-run"},
			wantErr: true,
		},
		{
			name: "verbose mode",
			args: []string{"--config", serverConfigPath, "--verbose", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Verbose mode enabled") {
					t.Error("output missing verbose mode notification")
				}
			},
		},
		{
			name: "json output",
			args: []string{"--config", serverConfigPath, "--json", "--dry-run"},
			validateOutput: func(t *testing.T, output string) {
				// Check if output is valid JSON
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("output is not valid JSON: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewTestCommand()
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

func TestScriptTestIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-scripttest-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple script test file
	scriptContent := `# Test script for MCP server
# Send initialize request
-> {"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
# Expect success response
<- {"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "*"}}}

# Send tools/list request
-> {"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
# Expect tools list in response
<- {"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "*"}]}}

# Send shutdown request
-> {"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": {}}
# Expect success response
<- {"jsonrpc": "2.0", "id": 3, "result": {"success": true}}
`

	scriptPath := filepath.Join(tempDir, "test_script.mcp")
	err = os.WriteFile(scriptPath, []byte(scriptContent), 0644)
	if err != nil {
		t.Fatalf("failed to write script file: %v", err)
	}

	// Create a test server configuration
	serverConfigJSON := `{
		"name": "script-test-server",
		"version": "1.0.0",
		"transport": "stdio",
		"command": "cat",
		"tools": [
			{
				"name": "tool1",
				"description": "A test tool",
				"schema": {
					"type": "object",
					"properties": {
						"foo": {"type": "string"}
					},
					"required": ["foo"]
				}
			}
		]
	}`

	serverConfigPath := filepath.Join(tempDir, "script_server.json")
	err = os.WriteFile(serverConfigPath, []byte(serverConfigJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write server config file: %v", err)
	}

	// Test running with script
	t.Run("run with script", func(t *testing.T) {
		cmd := NewTestCommand()
		args := []string{
			"--config", serverConfigPath,
			"--script", scriptPath,
			"--dry-run",
		}

		stdout, stderr, err := test.RunCommand(t, cmd, args)
		if err != nil {
			t.Errorf("Execute() error = %v\nStderr: %s", err, stderr)
			return
		}

		// Check that script was properly parsed
		if !strings.Contains(stdout, "Script test plan:") {
			t.Error("output missing script test plan")
		}

		// Check for expected messages
		expectedPhrases := []string{
			"initialize",
			"tools/list",
			"shutdown",
		}

		for _, phrase := range expectedPhrases {
			if !strings.Contains(stdout, phrase) {
				t.Errorf("output missing expected message %q", phrase)
			}
		}
	})

	// Test script syntax validation
	t.Run("script syntax validation", func(t *testing.T) {
		// Create an invalid script
		invalidScript := `# Invalid script
-> {"jsonrpc": "2.0", "id": 1, "method": "initialize"
# Missing closing bracket
<- {"jsonrpc": "2.0", "id": 1, "result": {}}
`

		invalidScriptPath := filepath.Join(tempDir, "invalid_script.mcp")
		err = os.WriteFile(invalidScriptPath, []byte(invalidScript), 0644)
		if err != nil {
			t.Fatalf("failed to write invalid script file: %v", err)
		}

		cmd := NewTestCommand()
		args := []string{
			"--config", serverConfigPath,
			"--script", invalidScriptPath,
			"--validate-only",
		}

		_, _, err := test.RunCommand(t, cmd, args)
		if err == nil {
			t.Error("expected error for invalid script, got nil")
		}
	})

	// Test multiple scripts
	t.Run("multiple scripts", func(t *testing.T) {
		// Create a second script
		script2Content := `# Second test script
-> {"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"name": "tool1", "params": {"foo": "bar"}}}
<- {"jsonrpc": "2.0", "id": 1, "result": {"success": true}}
`

		script2Path := filepath.Join(tempDir, "test_script2.mcp")
		err = os.WriteFile(script2Path, []byte(script2Content), 0644)
		if err != nil {
			t.Fatalf("failed to write second script file: %v", err)
		}

		cmd := NewTestCommand()
		args := []string{
			"--config", serverConfigPath,
			"--script", scriptPath,
			"--script", script2Path,
			"--dry-run",
		}

		stdout, stderr, err := test.RunCommand(t, cmd, args)
		if err != nil {
			t.Errorf("Execute() error = %v\nStderr: %s", err, stderr)
			return
		}

		// Check that both scripts were loaded
		if !strings.Contains(stdout, "Multiple script files") {
			t.Error("output missing multiple script files message")
		}

		// Check for scripts content
		if !strings.Contains(stdout, "tools/call") {
			t.Error("output missing content from second script")
		}
	})
}
