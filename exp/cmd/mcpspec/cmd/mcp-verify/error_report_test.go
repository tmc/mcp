package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestVerifyErrorReporting(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a test spec file with multiple methods
	specContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"methods": [
			{
				"name": "initialize",
				"params": {},
				"result": {
					"type": "object",
					"required": ["server_info", "features"]
				}
			},
			{
				"name": "shutdown",
				"params": {},
				"result": {
					"type": "object",
					"required": ["success"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording with multiple violations
	recordingContent := `# Name: Test Recording
# Server: Test Server
-> {"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}
<- {"jsonrpc": "2.0", "result": {"server_info": {"name": "Test Server", "version": "1.0.0"}}, "id": 1}
-> {"jsonrpc": "2.0", "method": "shutdown", "params": {}, "id": 2}
<- {"jsonrpc": "2.0", "result": {"status": "ok"}, "id": 2}
-> {"jsonrpc": "2.0", "method": "nonexistent", "params": {}, "id": 3}
<- {"jsonrpc": "2.0", "result": {}, "id": 3}
`
	recordingPath := h.WriteFile("test-recording.json", recordingContent)

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()
	cmd.continueOnError = true // Continue on errors to collect all errors

	// Execute with valid spec and recording
	err := cmd.Execute(context.Background(), []string{
		"-s", specPath,
		"-r", recordingPath,
		"--continue-on-error",
	})

	// Should get an error since there are multiple violations
	if err == nil {
		t.Fatalf("Expected error for spec violations, got none")
	}

	// Check output for detailed error report
	stderr := h.GetStderr()
	stdout := h.GetStdout()

	// Check for validation errors in stderr (with our new format)
	if !bytes.Contains([]byte(stderr), []byte("Validation error: required field")) {
		t.Errorf("Expected validation error message, got: %s", stderr)
	}

	// Check for method not found error in stderr
	if !bytes.Contains([]byte(stderr), []byte("method nonexistent not found in specification")) {
		t.Errorf("Expected method not found error, got: %s", stderr)
	}

	// Check for verification completed with errors summary in stdout
	if !bytes.Contains([]byte(stdout), []byte("Verification completed with")) {
		t.Errorf("Expected verification completion summary, got: %s", stdout)
	}
}

func TestVerifyErrorReportJSON(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a test spec file
	specContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"methods": [
			{
				"name": "initialize",
				"params": {},
				"result": {
					"type": "object",
					"required": ["server_info", "features"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording with missing 'features' field
	recordingContent := `# Name: Test Recording
# Server: Test Server
-> {"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}
<- {"jsonrpc": "2.0", "result": {"server_info": {"name": "Test Server", "version": "1.0.0"}}, "id": 1}
`
	recordingPath := h.WriteFile("test-recording.json", recordingContent)

	// Create a report output file
	reportPath := filepath.Join(h.TempDir(), "report.json")

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()

	// Execute with valid spec and recording, requesting JSON report
	err := cmd.Execute(context.Background(), []string{
		"-s", specPath,
		"-r", recordingPath,
		"-f", "json",
		"-o", reportPath,
	})

	// Should get an error due to missing 'features' field
	if err == nil {
		t.Fatalf("Expected error for spec violation, got none")
	}

	// Check if the report file was created
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Expected report file to be created at %s", reportPath)
	}

	// Read and parse the report file
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	// Check if the report is valid JSON
	var report map[string]interface{}
	if err := json.Unmarshal(reportData, &report); err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}

	// Check if the report contains validation errors
	if errorsArr, ok := report["errors"].([]interface{}); !ok || len(errorsArr) == 0 {
		t.Errorf("Expected errors array in JSON report")
	}
}

func TestVerifyErrorLocation(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a test spec file
	specContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"methods": [
			{
				"name": "initialize",
				"params": {},
				"result": {
					"type": "object",
					"required": ["server_info", "features"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording with missing 'features' field
	recordingContent := `# Name: Test Recording
# Server: Test Server
-> {"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}
<- {"jsonrpc": "2.0", "result": {"server_info": {"name": "Test Server", "version": "1.0.0"}}, "id": 1}
`
	recordingPath := h.WriteFile("test-recording.json", recordingContent)

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()

	// Execute with valid spec and recording
	err := cmd.Execute(context.Background(), []string{
		"-s", specPath,
		"-r", recordingPath,
		"--verbose",
	})

	// Should get an error due to missing 'features' field
	if err == nil {
		t.Fatalf("Expected error for spec violation, got none")
	}

	// Check output for error location information
	stderr := h.GetStderr()

	// Check for line number information in stderr
	if !strings.Contains(stderr, "line") && !strings.Contains(stderr, "at") {
		t.Errorf("Expected error location information, got: %s", stderr)
	}
}
