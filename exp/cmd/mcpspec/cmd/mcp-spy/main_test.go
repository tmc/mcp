package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tmc/mcp/cmd/mcpspec/internal/test"
)

func TestTrafficMonitoring(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-spy-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test input file with sample JSON-RPC messages
	inputData := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
{"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "test-server", "version": "1.0.0"}}}
{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
{"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "tool1", "description": "A test tool"}]}}
{"jsonrpc": "2.0", "id": 3, "method": "shutdown", "params": {}}
{"jsonrpc": "2.0", "id": 3, "result": {"success": true}}
`
	inputPath := filepath.Join(tempDir, "input.json")
	err = os.WriteFile(inputPath, []byte(inputData), 0644)
	if err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// Test cases for traffic monitoring
	testCases := []struct {
		name           string
		args           []string
		wantErr        bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name: "basic traffic monitoring",
			args: []string{"--input", inputPath},
			validateOutput: func(t *testing.T, output string) {
				// Check for request/response formatting
				if !strings.Contains(output, "→ REQUEST") {
					t.Error("output missing request formatting")
				}
				if !strings.Contains(output, "← RESPONSE") {
					t.Error("output missing response formatting")
				}
			},
		},
		{
			name: "colorized output",
			args: []string{"--input", inputPath, "--color"},
			validateOutput: func(t *testing.T, output string) {
				// Check for ANSI color codes
				if !strings.Contains(output, "\x1b[") {
					t.Error("output missing ANSI color codes")
				}
			},
		},
		{
			name: "raw mode",
			args: []string{"--input", inputPath, "--raw"},
			validateOutput: func(t *testing.T, output string) {
				// Check for unformatted JSON
				if !strings.Contains(output, `{"jsonrpc":"2.0"`) {
					t.Error("output missing raw JSON")
				}
				if strings.Contains(output, "→ REQUEST") {
					t.Error("raw mode should not include formatting")
				}
			},
		},
		{
			name: "filter by method",
			args: []string{"--input", inputPath, "--method", "initialize"},
			validateOutput: func(t *testing.T, output string) {
				// Should only show initialize messages
				if !strings.Contains(output, "initialize") {
					t.Error("output missing initialize method")
				}
				if strings.Contains(output, "tools/list") {
					t.Error("output should not include tools/list method")
				}
			},
		},
		{
			name: "filter by id",
			args: []string{"--input", inputPath, "--id", "2"},
			validateOutput: func(t *testing.T, output string) {
				// Should only show messages with ID 2
				if strings.Contains(output, `"id":1`) {
					t.Error("output should not include messages with ID 1")
				}
				if !strings.Contains(output, `"id":2`) {
					t.Error("output missing messages with ID 2")
				}
			},
		},
		{
			name: "show timing information",
			args: []string{"--input", inputPath, "--timing"},
			validateOutput: func(t *testing.T, output string) {
				// Check for timing information
				if !strings.Contains(output, "Time:") {
					t.Error("output missing timing information")
				}
			},
		},
		{
			name: "output to file",
			args: []string{"--input", inputPath, "--output", filepath.Join(tempDir, "output.json")},
			validateOutput: func(t *testing.T, output string) {
				// Output should be empty (written to file)
				if output != "" {
					t.Errorf("expected empty stdout, got: %q", output)
				}

				// Check that output file exists
				outputPath := filepath.Join(tempDir, "output.json")
				_, err := os.Stat(outputPath)
				if os.IsNotExist(err) {
					t.Errorf("output file not created: %v", err)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewSpyCommand()
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

func TestSessionRecording(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "mcpspec-spy-session-*")
	if err != nil {
		t.Fatalf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test input file with sample JSON-RPC messages
	inputData := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}
{"jsonrpc": "2.0", "id": 1, "result": {"serverInfo": {"name": "test-server", "version": "1.0.0"}}}
{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}}
{"jsonrpc": "2.0", "id": 2, "result": {"tools": [{"name": "tool1", "description": "A test tool"}]}}
`
	inputPath := filepath.Join(tempDir, "session_input.json")
	err = os.WriteFile(inputPath, []byte(inputData), 0644)
	if err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// Test session recording
	t.Run("record session", func(t *testing.T) {
		recordPath := filepath.Join(tempDir, "session.mcp")
		cmd := NewSpyCommand()
		args := []string{
			"--input", inputPath,
			"--record", recordPath,
		}

		stdout, stderr, err := test.RunCommand(t, cmd, args)
		if err != nil {
			t.Errorf("Execute() error = %v\nStderr: %s", err, stderr)
			return
		}

		// Check that session file was created
		_, err = os.Stat(recordPath)
		if os.IsNotExist(err) {
			t.Errorf("session file not created: %v", err)
			return
		}

		// Check that stdout still contains the traffic
		if !strings.Contains(stdout, "initialize") {
			t.Error("output missing initialize method")
		}

		// Check that session file contains the traffic in script format
		sessionData, err := os.ReadFile(recordPath)
		if err != nil {
			t.Errorf("failed to read session file: %v", err)
			return
		}

		sessionContent := string(sessionData)
		if !strings.Contains(sessionContent, "->") {
			t.Error("session file missing request markers (->)")
		}
		if !strings.Contains(sessionContent, "<-") {
			t.Error("session file missing response markers (<-)")
		}
	})

	// Test session metadata
	t.Run("session metadata", func(t *testing.T) {
		metadataPath := filepath.Join(tempDir, "session_with_metadata.mcp")
		cmd := NewSpyCommand()
		args := []string{
			"--input", inputPath,
			"--record", metadataPath,
			"--name", "Test Session",
			"--description", "A test session recording",
		}

		_, _, err := test.RunCommand(t, cmd, args)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}

		// Check that session file was created
		sessionData, err := os.ReadFile(metadataPath)
		if err != nil {
			t.Errorf("failed to read session file: %v", err)
			return
		}

		// Check for metadata in the session file
		sessionContent := string(sessionData)
		if !strings.Contains(sessionContent, "# Name: Test Session") {
			t.Error("session file missing name metadata")
		}
		if !strings.Contains(sessionContent, "# Description: A test session recording") {
			t.Error("session file missing description metadata")
		}
		if !strings.Contains(sessionContent, "# Date:") {
			t.Error("session file missing date metadata")
		}
	})

	// Test session with filtering
	t.Run("filtered session recording", func(t *testing.T) {
		filteredPath := filepath.Join(tempDir, "filtered_session.mcp")
		cmd := NewSpyCommand()
		args := []string{
			"--input", inputPath,
			"--record", filteredPath,
			"--method", "initialize",
		}

		_, _, err := test.RunCommand(t, cmd, args)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
			return
		}

		// Check that filtered session file was created
		sessionData, err := os.ReadFile(filteredPath)
		if err != nil {
			t.Errorf("failed to read session file: %v", err)
			return
		}

		// Check that only initialize messages are included
		sessionContent := string(sessionData)
		if !strings.Contains(sessionContent, "initialize") {
			t.Error("session file missing initialize method")
		}
		if strings.Contains(sessionContent, "tools/list") {
			t.Error("session file should not include tools/list method")
		}
	})
}
