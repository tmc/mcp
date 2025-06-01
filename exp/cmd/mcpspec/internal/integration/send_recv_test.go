package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/jsonrpc"
	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestSendReceiveIntegration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test cases for different types of messages
	testCases := []struct {
		name         string
		sendArgs     []string
		sendInput    string
		recvArgs     []string
		validateRecv func(t *testing.T, output string)
	}{
		{
			name:      "request message",
			sendArgs:  []string{"test.method"},
			sendInput: `{"foo": "bar", "baz": 42}`,
			recvArgs:  []string{},
			validateRecv: func(t *testing.T, output string) {
				if !strings.Contains(output, "Request: test.method") {
					t.Errorf("output missing request method: %q", output)
				}
				if !strings.Contains(output, "\"foo\": \"bar\"") {
					t.Errorf("output missing params foo: %q", output)
				}
				if !strings.Contains(output, "\"baz\": 42") {
					t.Errorf("output missing params baz: %q", output)
				}
			},
		},
		{
			name:      "notification message",
			sendArgs:  []string{"--notification", "test.notification"},
			sendInput: `{"event": "update", "data": {"status": "complete"}}`,
			recvArgs:  []string{},
			validateRecv: func(t *testing.T, output string) {
				if !strings.Contains(output, "Notification: test.notification") {
					t.Errorf("output missing notification method: %q", output)
				}
				if !strings.Contains(output, "\"event\": \"update\"") {
					t.Errorf("output missing params event: %q", output)
				}
			},
		},
		{
			name:      "extract field",
			sendArgs:  []string{"test.query"},
			sendInput: `{"nested": {"value": "target"}}`,
			recvArgs:  []string{"--field", "params.nested.value"},
			validateRecv: func(t *testing.T, output string) {
				if strings.TrimSpace(output) != "target" {
					t.Errorf("field extraction failed: got %q, want %q", output, "target")
				}
			},
		},
		{
			name:      "custom ID",
			sendArgs:  []string{"--id", "42", "test.method"},
			sendInput: `{}`,
			recvArgs:  []string{},
			validateRecv: func(t *testing.T, output string) {
				if !strings.Contains(output, "id: 42") {
					t.Errorf("output missing custom ID: %q", output)
				}
			},
		},
		{
			name:      "raw mode",
			sendArgs:  []string{"test.method"},
			sendInput: `{"foo": "bar"}`,
			recvArgs:  []string{"--raw"},
			validateRecv: func(t *testing.T, output string) {
				var msg jsonrpc.Message
				if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &msg); err != nil {
					t.Errorf("failed to parse raw output: %v", err)
				}
				if msg.Method != "test.method" {
					t.Errorf("raw output has wrong method: got %q, want %q", msg.Method, "test.method")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Files for this test case
			sendOutputFile := filepath.Join(tempDir, "send-output.json")

			// Run mcp-send
			sendCmd := main.NewSendCommand()
			sendArgs := append([]string{"--output", sendOutputFile}, tc.sendArgs...)
			_, _, err := test.RunCommandWithInput(t, sendCmd, sendArgs, tc.sendInput)
			if err != nil {
				t.Fatalf("mcp-send failed: %v", err)
			}

			// Verify send output file exists
			if _, err := os.Stat(sendOutputFile); os.IsNotExist(err) {
				t.Fatalf("send output file not created: %v", err)
			}

			// Run mcp-recv on the send output
			recvCmd := main.NewRecvCommand()
			recvArgs := append([]string{"--input", sendOutputFile}, tc.recvArgs...)
			stdout, _, err := test.RunCommand(t, recvCmd, recvArgs)
			if err != nil {
				t.Fatalf("mcp-recv failed: %v", err)
			}

			// Validate the recv output
			tc.validateRecv(t, stdout)
		})
	}
}

func TestSendReceivePipeline(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-pipeline-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create intermediate files
	pipelineFiles := map[string]string{
		"params.json":  `{"action": "query", "target": "users", "filter": {"active": true}}`,
		"request.json": "", // Will be created by mcp-send
	}

	// Create files in temp directory
	for name, content := range pipelineFiles {
		path := filepath.Join(tempDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file %s: %v", name, err)
		}
	}

	// Test pipeline:
	// 1. mcp-send creates a request message from params.json
	// 2. mcp-recv parses the request and extracts specific fields

	// Step 1: Run mcp-send to create request
	sendCmd := main.NewSendCommand()
	sendArgs := []string{
		"--params", filepath.Join(tempDir, "params.json"),
		"--output", filepath.Join(tempDir, "request.json"),
		"db.query",
	}

	_, _, err = test.RunCommand(t, sendCmd, sendArgs)
	if err != nil {
		t.Fatalf("mcp-send failed: %v", err)
	}

	// Step 2: Run mcp-recv to extract target field
	recvCmd := main.NewRecvCommand()
	recvArgs := []string{
		"--input", filepath.Join(tempDir, "request.json"),
		"--field", "params.target",
	}

	stdout, _, err := test.RunCommand(t, recvCmd, recvArgs)
	if err != nil {
		t.Fatalf("mcp-recv failed: %v", err)
	}

	// Validate the extracted field
	if strings.TrimSpace(stdout) != "users" {
		t.Errorf("field extraction failed: got %q, want %q", stdout, "users")
	}

	// Step 3: Run mcp-recv to extract filter
	recvArgsFilter := []string{
		"--input", filepath.Join(tempDir, "request.json"),
		"--field", "params.filter",
	}

	stdoutFilter, _, err := test.RunCommand(t, recvCmd, recvArgsFilter)
	if err != nil {
		t.Fatalf("mcp-recv filter extraction failed: %v", err)
	}

	// Validate the extracted filter object
	if !strings.Contains(stdoutFilter, "\"active\": true") {
		t.Errorf("filter extraction failed: %q", stdoutFilter)
	}
}
