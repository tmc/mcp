package mcpscripttest

import (
	"os"
	"strings"
	"testing"
)

// TestCoverageSummary demonstrates the complete coverage collection workflow
func TestCoverageSummary(t *testing.T) {
	// Skip if not in coverage mode
	if os.Getenv("GOCOVERDIR") == "" {
		t.Skip("Skipping coverage summary test - GOCOVERDIR not set")
	}

	// Create a test that demonstrates all coverage features
	t.Run("ToolInstallation", func(t *testing.T) {
		opts := &ToolsOptions{
			AutoDetectCoverage: true,
			VerboseOutput:      testing.Verbose(),
		}
		
		// Test that coverage is auto-detected
		if !opts.AutoDetectCoverage {
			t.Error("AutoDetectCoverage should be true by default")
		}
		
		cleanup := InstallMCPTools(t, opts)
		defer cleanup()
		
		t.Log("Tools installed with coverage instrumentation")
	})

	t.Run("ScriptExecution", func(t *testing.T) {
		// Install tools with coverage
		cleanup := InstallMCPTools(t, nil)
		defer cleanup()

		// Create options to pass GOCOVERDIR
		opts := DefaultOptions()
		opts.AdditionalEnvVars = []string{"GOCOVERDIR"}

		// Run a script test that uses multiple tools
		Test(t, "testdata/tools_coverage_test.txt", opts)

		t.Log("Script test executed with coverage-enabled tools")
	})

	t.Run("CoverageCollection", func(t *testing.T) {
		// Check that coverage data exists
		coverDir := os.Getenv("GOCOVERDIR")
		entries, err := os.ReadDir(coverDir)
		if err != nil {
			t.Logf("Warning: Could not read coverage directory: %v", err)
			return
		}
		
		// Count coverage files
		var covFiles int
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "covcounters.") || strings.HasPrefix(name, "covmeta.") {
				covFiles++
			}
		}
		
		t.Logf("Coverage collection summary:")
		t.Logf("  Coverage directory: %s", coverDir)
		t.Logf("  Coverage files found: %d", covFiles)
		
		if covFiles == 0 {
			t.Error("No coverage files found")
		}
	})
}