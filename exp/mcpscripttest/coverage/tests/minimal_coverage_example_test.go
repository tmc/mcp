package tests

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMinimalCoverageExample shows the simplest way to run a scripttest with coverage
func TestMinimalCoverageExample(t *testing.T) {
	// Step 1: Set up coverage directory (this enables coverage collection)
	coverDir := t.TempDir()
	t.Setenv("GOCOVERDIR", coverDir)

	// Step 2: Install coverage-enabled tools
	cleanup := mcpscripttest.InstallMCPTools(t, nil)
	defer cleanup()

	// Step 3: Run the scripttest (pass GOCOVERDIR to script environment)
	opts := mcpscripttest.DefaultOptions()
	opts.AdditionalEnvVars = []string{"GOCOVERDIR"}
	mcpscripttest.Test(t, "../../testdata/simple_coverage_demo.txt", opts)

	// Coverage data is now in coverDir
	t.Logf("Coverage data saved to: %s", coverDir)
}
