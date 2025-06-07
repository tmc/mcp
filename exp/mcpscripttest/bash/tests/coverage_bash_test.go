package tests

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestBashCoverage(t *testing.T) {
	// Test bash coverage collection
	mcpscripttest.Test(t, "../../testdata/bash_coverage_test.txt")
}

func TestBashCoverageSimple(t *testing.T) {
	// Simplified test for debugging
	mcpscripttest.Test(t, "../../testdata/bash_coverage_simple_test.txt")
}

func TestBashCoverageIntegration(t *testing.T) {
	// Note: testcallgraph tool is available as prebuilt binary testcallgraph-tool
	// Skip installation and run test directly

	// Test integration with testcallgraph
	mcpscripttest.Test(t, "../../testdata/testcallgraph_bash_test.txt")
}

func TestBashTxtarIntegration(t *testing.T) {
	// Test bash scripts in txtar files
	mcpscripttest.Test(t, "../../testdata/bash_txtar_test.txt")
}
