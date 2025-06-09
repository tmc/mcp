package tests

import (
	"os"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// TestControlledCoverage demonstrates running scripttest with controlled coverage collection
func TestControlledCoverage(t *testing.T) {
	// Create a single coverage directory that we control
	coverDir := t.TempDir()

	// Save original GOCOVERDIR
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)

	// Set our controlled coverage directory
	os.Setenv("GOCOVERDIR", coverDir)

	// Install tools - they will use our coverage directory
	cleanup := mcpscripttest.InstallMCPTools(t, nil)
	defer cleanup()

	// Create options that pass GOCOVERDIR but don't let it propagate further
	opts := mcpscripttest.DefaultOptions()
	opts.AdditionalEnvVars = []string{"GOCOVERDIR"}

	// Run the test
	mcpscripttest.Test(t, "../../testdata/simple_coverage_demo.txt", opts)

	// Now all coverage should be in our controlled directory
	entries, err := os.ReadDir(coverDir)
	if err != nil {
		t.Fatalf("Failed to read coverage directory: %v", err)
	}

	// Count coverage files
	covFiles := 0
	for _, entry := range entries {
		name := entry.Name()
		if len(name) > 3 && (name[:3] == "cov") {
			covFiles++
			t.Logf("Coverage file: %s", name)
		}
	}

	t.Logf("Total coverage files in controlled directory: %d", covFiles)
	t.Logf("Coverage directory: %s", coverDir)
}
