package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestVerifyUsage(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()

	// Execute with help flag
	err := cmd.Execute(context.Background(), []string{"--help"})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that usage is printed
	stdout := h.GetStdout()
	if !bytes.Contains([]byte(stdout), []byte("Usage: mcp-verify")) {
		t.Errorf("Expected usage to be printed, got %s", stdout)
	}
}

func TestVerifyRequiredFlags(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()

	// Execute without required flags
	err := cmd.Execute(context.Background(), []string{})
	if err == nil {
		t.Fatalf("Expected error for missing required flags, got none")
	}
}

func TestVerifySpecFileValidation(t *testing.T) {
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
					"required": ["server_info"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording file
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
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check output for successful verification
	stdout := h.GetStdout()
	if !bytes.Contains([]byte(stdout), []byte("Verification successful")) {
		t.Errorf("Expected verification successful message, got: %s", stdout)
	}
}

func TestVerifySpecViolation(t *testing.T) {
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
	})

	// Should get an error due to missing 'features' field
	if err == nil {
		t.Fatalf("Expected error for spec violation, got none")
	}

	// Check output for verification error
	stderr := h.GetStderr()
	if !bytes.Contains([]byte(stderr), []byte("Validation error: required field")) {
		t.Errorf("Expected validation error message, got: %s", stderr)
	}
}

func TestVerifyMultipleRecordings(t *testing.T) {
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
					"required": ["server_info"]
				}
			},
			{
				"name": "shutdown",
				"params": {},
				"result": {
					"type": "object"
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create test recording directory
	recordingDirPath := filepath.Join(h.TempDir(), "recordings")
	os.Mkdir(recordingDirPath, 0755)

	// Create recording file 1
	recording1Content := `# Name: Test Recording 1
# Server: Test Server
-> {"jsonrpc": "2.0", "method": "initialize", "params": {}, "id": 1}
<- {"jsonrpc": "2.0", "result": {"server_info": {"name": "Test Server", "version": "1.0.0"}}, "id": 1}
`
	recording1Path := filepath.Join(recordingDirPath, "recording1.json")
	os.WriteFile(recording1Path, []byte(recording1Content), 0644)

	// Create recording file 2
	recording2Content := `# Name: Test Recording 2
# Server: Test Server
-> {"jsonrpc": "2.0", "method": "shutdown", "params": {}, "id": 1}
<- {"jsonrpc": "2.0", "result": {}, "id": 1}
`
	recording2Path := filepath.Join(recordingDirPath, "recording2.json")
	os.WriteFile(recording2Path, []byte(recording2Content), 0644)

	// Create a new verify command
	cmd := NewVerifyCommand()
	cmd.Output = h.GetStdoutWriter()
	cmd.Error = h.GetStderrWriter()

	// Execute with directory of recordings
	err := cmd.Execute(context.Background(), []string{
		"-s", specPath,
		"-r", recordingDirPath,
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check output for successful verification
	stdout := h.GetStdout()
	if !bytes.Contains([]byte(stdout), []byte("Verification successful")) {
		t.Errorf("Expected verification successful message, got: %s", stdout)
	}
}

func TestVerifyJSONSchema(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a test spec file with JSON Schema
	specContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"methods": [
			{
				"name": "initialize",
				"params": {},
				"result": {
					"type": "object",
					"properties": {
						"server_info": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"version": {"type": "string"}
							},
							"required": ["name", "version"]
						}
					},
					"required": ["server_info"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording
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
		"--schema-validation",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check output for successful verification
	stdout := h.GetStdout()
	if !bytes.Contains([]byte(stdout), []byte("Verification successful")) {
		t.Errorf("Expected verification successful message, got: %s", stdout)
	}
}

func TestVerifyJSONSchemaViolation(t *testing.T) {
	// Setup test harness
	h := test.NewHarness(t)
	h.Setup()
	defer h.Teardown()

	// Create a test spec file with JSON Schema
	specContent := `{
		"name": "Test Server",
		"version": "1.0.0",
		"methods": [
			{
				"name": "initialize",
				"params": {},
				"result": {
					"type": "object",
					"properties": {
						"server_info": {
							"type": "object",
							"properties": {
								"name": {"type": "string"},
								"version": {"type": "number"}
							},
							"required": ["name", "version"]
						}
					},
					"required": ["server_info"]
				}
			}
		]
	}`
	specPath := h.WriteFile("test-spec.json", specContent)

	// Create a test recording with incorrect type for version
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
		"--schema-validation",
	})

	// Should get an error due to incorrect type for version
	if err == nil {
		t.Fatalf("Expected error for schema violation, got none")
	}

	// Check output for verification error
	stderr := h.GetStderr()
	if !bytes.Contains([]byte(stderr), []byte("schema validation error")) {
		t.Errorf("Expected schema validation error message, got: %s", stderr)
	}
}
