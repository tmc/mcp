package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPSpy runs all script tests for mcpspy
func TestMCPSpy(t *testing.T) {
	// Run all tests in the testdata directory
	mcpscripttest.Test(t, "testdata/*.txt")
}

// TestMCPSpyJSONScanner specifically tests JSON scanner functionality
func TestMCPSpyJSONScanner(t *testing.T) {
	// Run tests in the scripts subdirectory
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
