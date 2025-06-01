package main

import (
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestRecvCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		input    string
		wantErr  bool
		validate func(t *testing.T, stdout string, stderr string)
	}{
		{
			name:  "parse request",
			args:  []string{},
			input: `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if !strings.Contains(stdout, "test.method") {
					t.Errorf("output missing method: %q", stdout)
				}
				if !strings.Contains(stdout, "\"foo\": \"bar\"") {
					t.Errorf("output missing params: %q", stdout)
				}
				if !strings.Contains(stdout, "id: 1") {
					t.Errorf("output missing id: %q", stdout)
				}
			},
		},
		{
			name:  "parse notification",
			args:  []string{},
			input: `{"jsonrpc": "2.0", "method": "test.notification", "params": {"foo": "bar"}}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if !strings.Contains(stdout, "test.notification") {
					t.Errorf("output missing method: %q", stdout)
				}
				if !strings.Contains(stdout, "notification") {
					t.Errorf("output missing notification indicator: %q", stdout)
				}
			},
		},
		{
			name:  "parse response",
			args:  []string{},
			input: `{"jsonrpc": "2.0", "result": {"success": true}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if !strings.Contains(stdout, "\"success\": true") {
					t.Errorf("output missing result: %q", stdout)
				}
				if !strings.Contains(stdout, "id: 1") {
					t.Errorf("output missing id: %q", stdout)
				}
			},
		},
		{
			name:  "parse error response",
			args:  []string{},
			input: `{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if !strings.Contains(stdout, "Invalid Request") {
					t.Errorf("output missing error message: %q", stdout)
				}
				if !strings.Contains(stdout, "code: -32600") {
					t.Errorf("output missing error code: %q", stdout)
				}
			},
		},
		{
			name:    "invalid json",
			args:    []string{},
			input:   `{"jsonrpc": "2.0", "method": "test.method"`,
			wantErr: true,
		},
		{
			name:  "raw mode",
			args:  []string{"--raw"},
			input: `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				// Raw mode should output the original JSON exactly
				input := `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`
				if strings.TrimSpace(stdout) != input {
					t.Errorf("raw output doesn't match input: got %q, want %q", stdout, input)
				}
			},
		},
		{
			name:  "extract field",
			args:  []string{"--field", "method"},
			input: `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if strings.TrimSpace(stdout) != "test.method" {
					t.Errorf("field extraction failed: got %q, want %q", stdout, "test.method")
				}
			},
		},
		{
			name:  "extract nested field",
			args:  []string{"--field", "params.foo"},
			input: `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if strings.TrimSpace(stdout) != "bar" {
					t.Errorf("nested field extraction failed: got %q, want %q", stdout, "bar")
				}
			},
		},
		{
			name:    "extract non-existent field",
			args:    []string{"--field", "nonexistent"},
			input:   `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			wantErr: true,
		},
		{
			name:  "pretty print",
			args:  []string{"--pretty"},
			input: `{"jsonrpc": "2.0", "method": "test.method", "params": {"foo": "bar"}, "id": 1}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				if !strings.Contains(stdout, "\n") {
					t.Errorf("output is not pretty-printed: %q", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRecvCommand()
			stdout, stderr, err := test.RunCommandWithInput(t, cmd, tt.args, tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, stdout, stderr)
			}
		})
	}
}

func TestRecvCommandWithFiles(t *testing.T) {
	// Create temporary files for testing file-based input/output
	files := map[string]string{
		"input.json": `{"jsonrpc": "2.0", "method": "test.fileMethod", "params": {"foo": "bar"}, "id": 42}`,
		"output.txt": "",
	}

	// Test reading from a file
	cmd := NewRecvCommand()
	stdout, stderr, tempDir, err := test.RunCommandWithFiles(
		t,
		cmd,
		[]string{"-i", tempDir + "/input.json", "-o", tempDir + "/output.txt"},
		files,
	)

	if err != nil {
		t.Errorf("Execute() error = %v", err)
		return
	}

	// Output should be written to file, not stdout
	if stdout != "" {
		t.Errorf("expected empty stdout, got: %q", stdout)
	}

	// Read the output file
	outputPath := tempDir + "/output.txt"
	outputBytes, err := test.ReadFile(t, outputPath)
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}

	// Verify the output contains the expected information
	output := string(outputBytes)
	if !strings.Contains(output, "test.fileMethod") {
		t.Errorf("output file missing method: %q", output)
	}
	if !strings.Contains(output, "id: 42") {
		t.Errorf("output file missing id: %q", output)
	}
}
