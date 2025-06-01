package mcpscripttest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolsCoverageIntegration(t *testing.T) {
	// Create a temporary directory for coverage
	tmpDir, err := os.MkdirTemp("", "mcp-tools-coverage-integration")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original GOCOVERDIR
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)

	// Set temporary GOCOVERDIR
	coverDir := filepath.Join(tmpDir, "coverage")
	os.MkdirAll(coverDir, 0755)
	os.Setenv("GOCOVERDIR", coverDir)

	// Create tools options with auto-detection enabled
	toolOpts := &ToolsOptions{
		AutoDetectCoverage: true,
		ToolsDir:           filepath.Join(tmpDir, "tools"),
		Tools:              []string{"mcpdiff"},
		VerboseOutput:      testing.Verbose(),
	}

	// Install tools - should auto-detect coverage
	cleanup := InstallMCPTools(t, toolOpts)
	defer cleanup()

	// Create options to pass GOCOVERDIR to script test
	scriptOpts := DefaultOptions()
	scriptOpts.AdditionalEnvVars = []string{"GOCOVERDIR"}

	// Run the script test that uses mcpdiff
	Test(t, "testdata/tools_coverage_test.txt", scriptOpts)

	// Check if coverage data was generated
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Logf("Warning: Could not read coverage directory: %v", err)
		return
	}

	covFileCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if len(name) > 11 && name[:11] == "covcounters" || name[:7] == "covmeta" {
			covFileCount++
		}
	}

	if covFileCount == 0 {
		t.Logf("Warning: No coverage files found in %s", coverDir)
		t.Logf("Tool may not have been executed or coverage collection failed")
	} else {
		t.Logf("Found %d coverage files in %s", covFileCount, coverDir)
	}
}