package main

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// TestMCPShadowScripts runs all scripts in the testdata directory
func TestMCPShadowScripts(t *testing.T) {
	// Run all scripts
	mcpscripttest.Test(t, "testdata/*.txt")
}

// TestCompareModeSpecifically tests the compare mode functionality
func TestCompareModeSpecifically(t *testing.T) {
	// Use a specific script file for this test
	mcpscripttest.Test(t, "testdata/compare-mode.txt")
}
