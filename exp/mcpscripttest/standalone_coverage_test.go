package mcpscripttest

import (
	"os"
	"testing"
)

// TestStandaloneCoverage demonstrates running just a scripttest with coverage
func TestStandaloneCoverage(t *testing.T) {
	// Create a temporary directory for coverage
	coverDir := t.TempDir()
	
	// Set GOCOVERDIR - this triggers automatic coverage instrumentation
	t.Setenv("GOCOVERDIR", coverDir)
	
	// Install tools with coverage (auto-detected from GOCOVERDIR)
	cleanup := InstallMCPTools(t, nil)
	defer cleanup()
	
	// Create options to pass GOCOVERDIR to the script environment
	opts := DefaultOptions()
	opts.AdditionalEnvVars = []string{"GOCOVERDIR"}
	
	// Run just the scripttest file
	Test(t, "testdata/tools_coverage_test.txt", opts)
	
	// Show coverage results
	t.Logf("Coverage collected in: %s", coverDir)
	
	// List coverage files
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Fatalf("Failed to read coverage directory: %v", err)
	}
	
	for _, entry := range entries {
		t.Logf("Coverage file: %s", entry.Name())
	}
	
	// To view coverage after the test:
	t.Logf("To analyze coverage run:")
	t.Logf("  go tool covdata percent -i %s", coverDir)
}