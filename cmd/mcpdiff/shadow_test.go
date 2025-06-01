package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var verboseTest = flag.Bool("test.verbose", false, "enable verbose test output")

// createTempFile creates a temporary file with the given content
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	file, err := os.CreateTemp("", "mcpdiff-test-*.mcp")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("Failed to write content: %v", err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close file: %v", err)
	}

	return file.Name()
}

// runMcpdiff runs mcpdiff command with given arguments and returns output
func runMcpdiff(args ...string) (string, error) {
	cmd := exec.Command("go", append([]string{"run", "main.go"}, args...)...)
	cmd.Dir = "/Volumes/tmc/go/src/github.com/tmc/mcp/cmd/mcpdiff"

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Test comparing a single file with shadow records
func TestSingleFileWithShadow(t *testing.T) {
	t.Run("no differences", func(t *testing.T) {
		// Create a trace with identical primary and shadow responses
		content := `mcp-send {"jsonrpc":"2.0","method":"tools/list","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.000 spanid=resp1
mcp-send-shadow {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.100 spanid=shadow1 linksto=resp1
`

		tracePath := createTempFile(t, content)
		defer os.Remove(tracePath)

		// Run mcpdiff with single file (no -compare flag needed)
		arg := []string{tracePath}
		if *verboseTest {
			arg = append(arg, "-v")
		}

		output, err := runMcpdiff(arg...)
		if err != nil {
			t.Fatalf("mcpdiff failed: %v", err)
		}

		// Should find no differences
		if output == "" {
			t.Logf("No differences found - files match (as expected)")
		} else {
			t.Errorf("Expected no output, got: %s", output)
		}
	})

	t.Run("with differences", func(t *testing.T) {
		// Create a trace with different primary and shadow responses
		content := `mcp-send {"jsonrpc":"2.0","method":"tools/list","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.000 spanid=resp1
mcp-send-shadow {"jsonrpc":"2.0","result":["tool2"],"id":1} # 1001.100 spanid=shadow1 linksto=resp1
`

		tracePath := createTempFile(t, content)
		defer os.Remove(tracePath)

		// Run mcpdiff with single file
		arg := []string{tracePath}
		if *verboseTest {
			arg = append(arg, "-v")
		}

		output, err := runMcpdiff(arg...)
		if err == nil {
			t.Errorf("Expected mcpdiff to exit with status 1 for differences")
		}

		// Should show differences
		if strings.Contains(output, "send") && strings.Contains(output, "send-shadow") {
			t.Logf("Found comparison between primary and shadow: %s", output)
		} else {
			t.Errorf("Expected diff output, got: %s", output)
		}
	})

	t.Run("without shadow records", func(t *testing.T) {
		// Create a trace without shadow records
		content := `mcp-send {"jsonrpc":"2.0","method":"tools/list","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.000 spanid=resp1
`

		tracePath := createTempFile(t, content)
		defer os.Remove(tracePath)

		// Run mcpdiff with single file
		arg := []string{tracePath}
		if *verboseTest {
			arg = append(arg, "-v")
		}

		output, err := runMcpdiff(arg...)
		if err == nil {
			t.Errorf("Expected mcpdiff to exit with error for single file without shadow")
		}

		// Should error about no shadow records
		if strings.Contains(output, "no shadow records found") {
			t.Logf("Got expected error: %s", output)
		} else {
			t.Errorf("Expected error about no shadow records, got: %s", output)
		}
	})
}

// Test backward compatibility with -compare flag
func TestCompareFlag(t *testing.T) {
	// Create a trace with both primary and shadow responses
	content := `mcp-send {"jsonrpc":"2.0","method":"tools/list","id":1} # 1000.000 spanid=req1
mcp-send {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.000 spanid=resp1
mcp-send-shadow {"jsonrpc":"2.0","result":["tool1"],"id":1} # 1001.100 spanid=shadow1 linksto=resp1
`

	tracePath := createTempFile(t, content)
	defer os.Remove(tracePath)

	// Run mcpdiff with -compare flag (should still work)
	arg := []string{"-compare", tracePath}
	if *verboseTest {
		arg = append(arg, "-v")
	}

	output, err := runMcpdiff(arg...)
	if err != nil {
		t.Fatalf("mcpdiff failed: %v", err)
	}

	// Should work the same way as without the flag
	if output == "" {
		t.Logf("No differences found - files match (as expected)")
	} else {
		t.Errorf("Expected no output, got: %s", output)
	}
}
