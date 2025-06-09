package tests

import (
	"os"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// TestPracticalCoverageExample shows a practical example of running scripttest with coverage
func TestPracticalCoverageExample(t *testing.T) {
	// This is the simplest way to run a scripttest with coverage collection

	// 1. Create and set a coverage directory
	coverDir := t.TempDir()
	origCoverDir := os.Getenv("GOCOVERDIR")
	defer os.Setenv("GOCOVERDIR", origCoverDir)
	os.Setenv("GOCOVERDIR", coverDir)

	// 2. Install coverage-enabled tools (auto-detects GOCOVERDIR)
	cleanup := mcpscripttest.InstallMCPTools(t, nil)
	defer cleanup()

	// 3. Run scripttest with GOCOVERDIR in environment
	opts := mcpscripttest.DefaultOptions()
	opts.AdditionalEnvVars = []string{"GOCOVERDIR"}
	mcpscripttest.Test(t, "../../testdata/simple_coverage_demo.txt", opts)

	// 4. Report what was collected
	entries, _ := os.ReadDir(coverDir)
	covFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if len(name) > 3 && name[:3] == "cov" {
				covFiles++
			}
		}
	}

	t.Logf("Coverage collection complete:")
	t.Logf("  Directory: %s", coverDir)
	t.Logf("  Files collected: %d", covFiles)
	t.Logf("  To analyze: go tool covdata percent -i %s", coverDir)
}
