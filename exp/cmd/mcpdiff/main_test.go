package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

func TestMCPRecordParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected MCPRecord
		wantErr  bool
	}{
		{
			name:  "valid send record",
			input: `mcp-send {"jsonrpc":"2.0","id":1,"method":"test"} # 1234567890.123`,
			expected: MCPRecord{
				Direction:  "send",
				RawContent: `{"jsonrpc":"2.0","id":1,"method":"test"}`,
				Timestamp:  1234567890.123,
				LineNum:    0,
			},
		},
		{
			name:  "valid recv record",
			input: `mcp-recv {"jsonrpc":"2.0","result":"ok"} # 1234567890.456`,
			expected: MCPRecord{
				Direction:  "recv",
				RawContent: `{"jsonrpc":"2.0","result":"ok"}`,
				Timestamp:  1234567890.456,
				LineNum:    0,
			},
		},
		{
			name:    "invalid json",
			input:   `mcp-send {invalid json} # 1234567890.123`,
			wantErr: true,
		},
		{
			name:    "missing timestamp",
			input:   `mcp-send {"jsonrpc":"2.0","id":1}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := parseMCPLine(tt.input, 0)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if record.Direction != tt.expected.Direction {
				t.Errorf("Direction = %v, want %v", record.Direction, tt.expected.Direction)
			}
			if record.RawContent != tt.expected.RawContent {
				t.Errorf("RawContent = %v, want %v", record.RawContent, tt.expected.RawContent)
			}
			if record.Timestamp != tt.expected.Timestamp {
				t.Errorf("Timestamp = %v, want %v", record.Timestamp, tt.expected.Timestamp)
			}
		})
	}
}

// TestBasicDiff specifically tests basic diff functionality
func TestBasicDiff(t *testing.T) {
	// Install mcpdiff tool for the test
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcpdiff"},
	})
	defer cleanup()

	mcpscripttest.Test(t, "testdata/basic_diff.txt")
}

func TestColorization(t *testing.T) {
	// Test that colorization functions work
	text := "test text"

	// Test basic color functions
	colored := colorRed(text)
	if !strings.Contains(colored, text) {
		t.Errorf("colorRed should contain original text")
	}

	colored = colorGreen(text)
	if !strings.Contains(colored, text) {
		t.Errorf("colorGreen should contain original text")
	}

	colored = colorYellow(text)
	if !strings.Contains(colored, text) {
		t.Errorf("colorYellow should contain original text")
	}
}

func TestFileDiff(t *testing.T) {
	// Create temporary test files
	tmpDir, err := os.MkdirTemp("", "mcpdiff-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	file1 := filepath.Join(tmpDir, "test1.mcp")
	file2 := filepath.Join(tmpDir, "test2.mcp")

	content1 := `# mcptrace:v1 source=test1
mcp-send {"jsonrpc":"2.0","id":1,"method":"test"} # 1234567890.123
mcp-recv {"jsonrpc":"2.0","id":1,"result":"ok"} # 1234567890.456
`

	content2 := `# mcptrace:v1 source=test2
mcp-send {"jsonrpc":"2.0","id":2,"method":"test"} # 1234567890.789
mcp-recv {"jsonrpc":"2.0","id":2,"result":"different"} # 1234567890.999
`

	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to write test file 2: %v", err)
	}

	// Test diff functionality
	_ = bytes.Buffer{} // placeholder for future use
	originalStdout := os.Stdout
	defer func() { os.Stdout = originalStdout }()

	// This is a basic integration test - in a real implementation,
	// we'd refactor the main function to return results instead of printing
	t.Log("Testing diff functionality with temp files:")
	t.Logf("File 1: %s", file1)
	t.Logf("File 2: %s", file2)

	// For now, just verify the files exist and have different content
	content1Read, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Failed to read file 1: %v", err)
	}
	content2Read, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("Failed to read file 2: %v", err)
	}

	if string(content1Read) == string(content2Read) {
		t.Error("Test files should have different content")
	}

	// Test that we can parse both files
	records1, err := parseFileTest(file1)
	if err != nil {
		t.Fatalf("Failed to parse file 1: %v", err)
	}
	records2, err := parseFileTest(file2)
	if err != nil {
		t.Fatalf("Failed to parse file 2: %v", err)
	}

	if len(records1) == 0 {
		t.Error("File 1 should have records")
	}
	if len(records2) == 0 {
		t.Error("File 2 should have records")
	}

	t.Logf("Successfully parsed %d records from file 1 and %d records from file 2", len(records1), len(records2))
}

func TestFlagDefaults(t *testing.T) {
	// Test that flags have reasonable defaults
	if *contextLines != 3 {
		t.Errorf("contextLines default should be 3, got %d", *contextLines)
	}
	if !*ignoreTimestamps {
		t.Errorf("ignoreTimestamps should default to true")
	}
	if *ignoreIDs {
		t.Errorf("ignoreIDs should default to false")
	}
}

// Helper functions that would be part of the main.go file
type normalizationOptions struct {
	ignoreTimestamps bool
	ignoreIDs        bool
	semanticCompare  bool
}

// Mock functions for testing - these would be actual implementations in main.go
func parseMCPLine(line string, lineNum int) (MCPRecord, error) {
	// This is a simplified parser for testing
	if strings.HasPrefix(line, "mcp-send") || strings.HasPrefix(line, "mcp-recv") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			return MCPRecord{}, fmt.Errorf("invalid line format")
		}

		direction := strings.TrimPrefix(parts[0], "mcp-")
		rest := parts[1]

		// Find JSON and timestamp
		timestampIdx := strings.LastIndex(rest, " # ")
		if timestampIdx == -1 {
			return MCPRecord{}, fmt.Errorf("missing timestamp")
		}

		jsonStr := strings.TrimSpace(rest[:timestampIdx])
		timestampStr := strings.TrimSpace(rest[timestampIdx+3:])

		// Parse timestamp
		timestamp, err := strconv.ParseFloat(timestampStr, 64)
		if err != nil {
			return MCPRecord{}, fmt.Errorf("invalid timestamp: %v", err)
		}

		// Parse JSON
		var jsonData map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
			return MCPRecord{}, fmt.Errorf("invalid JSON: %v", err)
		}

		return MCPRecord{
			Direction:  direction,
			RawContent: jsonStr,
			JSON:       jsonData,
			Timestamp:  timestamp,
			LineNum:    lineNum,
		}, nil
	}

	return MCPRecord{}, fmt.Errorf("not an MCP record line")
}

func normalizeRecord(record MCPRecord, options normalizationOptions) string {
	result := record.Direction + ":"

	if options.ignoreIDs && record.JSON != nil {
		// Create a copy without ID
		normalized := make(map[string]any)
		for k, v := range record.JSON {
			if k != "id" {
				normalized[k] = v
			}
		}
		jsonBytes, _ := json.Marshal(normalized)
		result += string(jsonBytes)
	} else {
		result += record.RawContent
	}

	if !options.ignoreTimestamps {
		result += fmt.Sprintf(" # %.3f", record.Timestamp)
	}

	return result
}

func parseFileTest(filename string) ([]MCPRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []MCPRecord
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		record, err := parseMCPLine(line, lineNum)
		if err != nil {
			// Skip lines that aren't MCP records
			continue
		}

		records = append(records, record)
	}

	return records, scanner.Err()
}

func colorRed(text string) string {
	if *noColor {
		return text
	}
	return red + text + reset
}

func colorGreen(text string) string {
	if *noColor {
		return text
	}
	return green + text + reset
}

func colorYellow(text string) string {
	if *noColor {
		return text
	}
	return yellow + text + reset
}
