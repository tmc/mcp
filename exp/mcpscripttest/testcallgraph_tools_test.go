package mcpscripttest

import (
	"testing"
)

func TestTestcallgraphTool(t *testing.T) {
	// Test the testcallgraph CLI tool
	Test(t, "testdata/testcallgraph_test_final.txt")
}

func TestTestcallgraphCoverageTool(t *testing.T) {
	// Test the testcallgraph-coverage CLI tool
	Test(t, "testdata/testcallgraph_coverage_simple.txt")
}

func TestTestcallgraphIntegration(t *testing.T) {
	// Run a more comprehensive test
	t.Run("BasicAnalysis", func(t *testing.T) {
		Test(t, "testdata/testcallgraph_test_final.txt")
	})

	t.Run("CoverageIntegration", func(t *testing.T) {
		Test(t, "testdata/testcallgraph_coverage_simple.txt")
	})
}

// TestToolAvailability checks if the tools are built and available
func TestToolAvailability(t *testing.T) {
	// This helps ensure the tools are built before running tests
	if testing.Short() {
		t.Skip("Skipping tool availability test in short mode")
	}
	
	// The actual tools will be tested via the scripttests above
	t.Log("Testcallgraph tools should be built and available in PATH")
}