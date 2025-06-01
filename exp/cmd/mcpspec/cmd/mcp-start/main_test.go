package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestServerConfigParsing(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-start-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases for different server configurations
	testCases := []struct {
		name           string
		configJSON     string
		args           []string
		wantErr        bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "minimal valid config",
			configJSON: `{
				"name": "test-server",
				"version": "1.0.0",
				"transport": "stdio",
				"command": "echo Hello"
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
		{
			name: "missing required fields",
			configJSON: `{
				"name": "test-server"
			}`,
			args:    []string{"--config", "server.json"},
			wantErr: true,
		},
		{
			name: "http transport config",
			configJSON: `{
				"name": "http-server",
				"version": "1.0.0",
				"transport": "http",
				"port": 8080,
				"host": "localhost"
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
		{
			name: "websocket transport config",
			configJSON: `{
				"name": "websocket-server",
				"version": "1.0.0",
				"transport": "websocket",
				"port": 8080,
				"host": "localhost"
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
		{
			name: "stdio transport with command",
			configJSON: `{
				"name": "stdio-server",
				"version": "1.0.0",
				"transport": "stdio",
				"command": "echo 'Hello from server'",
				"environment": {
					"DEBUG": "true"
				}
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
		{
			name: "invalid transport type",
			configJSON: `{
				"name": "invalid-server",
				"version": "1.0.0",
				"transport": "invalid",
				"command": "echo Hello"
			}`,
			args:    []string{"--config", "server.json"},
			wantErr: true,
		},
		{
			name: "tool definitions",
			configJSON: `{
				"name": "tools-server",
				"version": "1.0.0",
				"transport": "stdio",
				"command": "echo Hello",
				"tools": [
					{
						"name": "tool1",
						"description": "Test tool 1",
						"schema": {
							"type": "object",
							"properties": {
								"foo": {"type": "string"}
							},
							"required": ["foo"]
						}
					},
					{
						"name": "tool2",
						"description": "Test tool 2",
						"schema": {
							"type": "object",
							"properties": {
								"bar": {"type": "number"}
							},
							"required": ["bar"]
						}
					}
				]
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
		{
			name: "with init params",
			configJSON: `{
				"name": "init-server",
				"version": "1.0.0",
				"transport": "stdio",
				"command": "echo Hello",
				"init_params": {
					"capabilities": ["tool1", "tool2"],
					"options": {"debug": true}
				}
			}`,
			args: []string{"--config", "server.json"},
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("expected output, got empty string")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a config file for this test case
			configPath := filepath.Join(tempDir, "server.json")
			err := os.WriteFile(configPath, []byte(tc.configJSON), 0644)
			if err != nil {
				t.Fatalf("failed to write config file: %v", err)
			}

			// Run mcp-start with the config
			cmd := NewStartCommand()

			// Replace the config file path with the absolute path
			for i, arg := range tc.args {
				if arg == "server.json" {
					tc.args[i] = configPath
				}
			}

			// Add the dry-run flag to avoid actually starting a server
			tc.args = append(tc.args, "--dry-run")

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

func TestServerLifecycleFunctions(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-lifecycle-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid server config
	configJSON := `{
		"name": "test-server",
		"version": "1.0.0",
		"transport": "stdio",
		"command": "echo '{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {\"success\": true}}'"
	}`

	configPath := filepath.Join(tempDir, "server.json")
	err = os.WriteFile(configPath, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Test starting the server
	t.Run("start and stop server", func(t *testing.T) {
		cmd := NewStartCommand()

		// Execute with timeout to ensure it doesn't run forever
		args := []string{"--config", configPath, "--timeout", "2"}
		stdout, stderr, err := test.RunCommand(t, cmd, args)

		// Since the real server would block until terminated, we expect a timeout or graceful exit
		// The important part is that we observe proper startup messages

		if stdout == "" {
			t.Error("expected output for server start, got empty string")
		}

		// Check for expected startup messages
		expectedPhrases := []string{
			"Starting MCP server",
			"test-server",
			"version 1.0.0",
		}

		for _, phrase := range expectedPhrases {
			if !containsString(stdout, phrase) && !containsString(stderr, phrase) {
				t.Errorf("expected output to contain %q, but it doesn't", phrase)
			}
		}
	})

	// Test server initialization
	t.Run("server initialization", func(t *testing.T) {
		initConfigJSON := `{
			"name": "init-test-server",
			"version": "1.0.0",
			"transport": "stdio",
			"command": "echo '{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {\"version\": \"1.0.0\", \"serverInfo\": {\"name\": \"test\"}}}' && sleep 1",
			"init_params": {
				"capabilities": ["tool1", "tool2"]
			}
		}`

		initConfigPath := filepath.Join(tempDir, "init-server.json")
		err = os.WriteFile(initConfigPath, []byte(initConfigJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cmd := NewStartCommand()
		args := []string{"--config", initConfigPath, "--timeout", "2"}
		stdout, stderr, err := test.RunCommand(t, cmd, args)

		// Check for initialization message
		if !containsString(stdout, "Initialized") && !containsString(stderr, "Initialized") {
			t.Error("expected output to contain initialization confirmation")
		}
	})

	// Test server shutdown
	t.Run("server shutdown", func(t *testing.T) {
		shutdownConfigJSON := `{
			"name": "shutdown-test-server",
			"version": "1.0.0",
			"transport": "stdio",
			"command": "echo '{\"jsonrpc\": \"2.0\", \"id\": 1, \"result\": {\"success\": true}}' && sleep 1"
		}`

		shutdownConfigPath := filepath.Join(tempDir, "shutdown-server.json")
		err = os.WriteFile(shutdownConfigPath, []byte(shutdownConfigJSON), 0644)
		if err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		cmd := NewStartCommand()
		args := []string{"--config", shutdownConfigPath, "--timeout", "2"}
		stdout, stderr, err := test.RunCommand(t, cmd, args)

		// Check for shutdown message
		if !containsString(stdout, "Shutting down") && !containsString(stderr, "Shutting down") {
			t.Error("expected output to contain shutdown message")
		}
	})
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return s != "" && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr || s[1:len(s)-1] != "" && containsString(s[1:len(s)-1], substr)))
}
