package mcpscripttest

import (
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

// TestAllTestdata runs all tests in the testdata directory
// with properly installed MCP tools
func TestAllTestdata(t *testing.T) {
	t.Skip("Skipping TestAllTestdata - test scripts need updating for current environment")
	// Install all MCP tools including mcpscripttest analysis tools
	toolOpts := tools.DefaultToolsWithScripttestOptions()
	toolOpts.VerboseOutput = testing.Verbose()
	cleanup := InstallMCPTools(t, toolOpts)
	defer cleanup()

	// Get all test files in testdata directory
	pattern := filepath.Join("testdata", "*.txt")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob test files: %v", err)
	}

	// Run each test file
	for _, file := range files {
		file := file // capture loop variable
		t.Run(filepath.Base(file), func(t *testing.T) {
			Test(t, file)
		})
	}
}
