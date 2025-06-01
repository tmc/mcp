package mcpscripttest

import (
	"testing"
)

func TestBashCoverage(t *testing.T) {
	// Test bash coverage collection
	Test(t, "testdata/bash_coverage_test.txt")
}

func TestBashCoverageSimple(t *testing.T) {
	// Simplified test for debugging
	Test(t, "testdata/bash_coverage_simple_test.txt")
}

func TestBashCoverageIntegration(t *testing.T) {
	// Install testcallgraph tool first
	cleanup := InstallMCPTools(t, &ToolsOptions{
		Tools: []string{"testcallgraph"},
	})
	defer cleanup()

	// Test integration with testcallgraph
	Test(t, "testdata/testcallgraph_bash_test.txt")
}

func TestBashTxtarIntegration(t *testing.T) {
	// Test bash scripts in txtar files
	Test(t, "testdata/bash_txtar_test.txt")
}