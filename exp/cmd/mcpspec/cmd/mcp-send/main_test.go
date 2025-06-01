package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestSendCommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		input    string
		wantErr  bool
		validate func(t *testing.T, stdout string, stderr string)
	}{
		{
			name:    "requires method",
			args:    []string{},
			wantErr: true,
		},
		{
			name:  "simple request",
			args:  []string{"test.method"},
			input: `{"foo": "bar"}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				var msg jsonrpc.Message
				if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&msg); err != nil {
					t.Errorf("invalid JSON output: %v", err)
					return
				}
				if msg.Version != "2.0" {
					t.Errorf("wrong JSON-RPC version: got %q, want %q", msg.Version, "2.0")
				}
				if msg.Method != "test.method" {
					t.Errorf("wrong method: got %q, want %q", msg.Method, "test.method")
				}
				if msg.ID != 1 {
					t.Errorf("wrong ID: got %v, want %v", msg.ID, 1)
				}

				var params map[string]interface{}
				if err := json.Unmarshal(msg.Params, &params); err != nil {
					t.Errorf("invalid params: %v", err)
					return
				}
				if v, ok := params["foo"]; !ok || v != "bar" {
					t.Errorf("wrong params: got %v, want {\"foo\": \"bar\"}", params)
				}
			},
		},
		{
			name:  "notification",
			args:  []string{"-n", "test.notification"},
			input: `{"foo": "bar"}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				var msg jsonrpc.Message
				if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&msg); err != nil {
					t.Errorf("invalid JSON output: %v", err)
					return
				}
				if msg.Version != "2.0" {
					t.Errorf("wrong JSON-RPC version: got %q, want %q", msg.Version, "2.0")
				}
				if msg.Method != "test.notification" {
					t.Errorf("wrong method: got %q, want %q", msg.Method, "test.notification")
				}
				if msg.ID != nil {
					t.Errorf("notification should not have ID: got %v", msg.ID)
				}
			},
		},
		{
			name:  "custom ID",
			args:  []string{"-i", "42", "test.method"},
			input: `{}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				var msg jsonrpc.Message
				if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&msg); err != nil {
					t.Errorf("invalid JSON output: %v", err)
					return
				}
				if msg.ID != 42 {
					t.Errorf("wrong ID: got %v, want %v", msg.ID, 42)
				}
			},
		},
		{
			name:  "pretty print",
			args:  []string{"--pretty", "test.method"},
			input: `{}`,
			validate: func(t *testing.T, stdout string, stderr string) {
				// Pretty-printed JSON should have newlines and indentation
				if !strings.Contains(stdout, "\n  ") {
					t.Errorf("output is not pretty-printed: %q", stdout)
				}

				var msg jsonrpc.Message
				if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&msg); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
			},
		},
		{
			name:  "null params",
			args:  []string{"test.method"},
			input: "",
			validate: func(t *testing.T, stdout string, stderr string) {
				var msg jsonrpc.Message
				if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&msg); err != nil {
					t.Errorf("invalid JSON output: %v", err)
					return
				}

				// When no params are provided, we should get null params (nil in the JSON)
				if string(msg.Params) != "null" && len(msg.Params) != 0 {
					t.Errorf("expected null params, got: %v", string(msg.Params))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSendCommand()
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

func TestSendCommandWithFiles(t *testing.T) {
	// Create temporary files for testing file-based params
	files := map[string]string{
		"params.json": `{"foo": "bar", "baz": 42}`,
		"output.json": "",
	}

	// Test writing to a file
	cmd := NewSendCommand()
	stdout, stderr, tempDir, err := test.RunCommandWithFiles(
		t,
		cmd,
		[]string{"-p", tempDir + "/params.json", "-o", tempDir + "/output.json", "test.fileMethod"},
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
	outputPath := tempDir + "/output.json"
	outputBytes, err := test.ReadFile(t, outputPath)
	if err != nil {
		t.Errorf("failed to read output file: %v", err)
		return
	}

	// Verify the output is valid JSON-RPC
	var msg jsonrpc.Message
	if err := json.Unmarshal(outputBytes, &msg); err != nil {
		t.Errorf("invalid JSON in output file: %v", err)
		return
	}

	// Check message properties
	if msg.Version != "2.0" {
		t.Errorf("wrong JSON-RPC version: got %q, want %q", msg.Version, "2.0")
	}
	if msg.Method != "test.fileMethod" {
		t.Errorf("wrong method: got %q, want %q", msg.Method, "test.fileMethod")
	}

	// Verify params were read from file correctly
	var params map[string]interface{}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		t.Errorf("invalid params: %v", err)
		return
	}
	if v, ok := params["foo"]; !ok || v != "bar" {
		t.Errorf("wrong params: missing or incorrect 'foo' value: %v", params)
	}
	if v, ok := params["baz"]; !ok || v != float64(42) {
		t.Errorf("wrong params: missing or incorrect 'baz' value: %v", params)
	}
}
