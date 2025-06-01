package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPShadowScripts runs all scripts in the testdata directory
func TestMCPShadowScripts(t *testing.T) {
	// Setup coverage environment automatically
	mcpscripttest.SetupCoverageEnvironment(t)

	// Run all scripts
	mcpscripttest.Test(t, "testdata/*.txt")
}

// TestCompareModeSpecifically tests the compare mode functionality
func TestCompareModeSpecifically(t *testing.T) {
	// Use a specific script file for this test
	mcpscripttest.Test(t, "testdata/compare-mode.txt")
}
