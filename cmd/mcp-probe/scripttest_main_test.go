package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPProbe runs all script tests for mcp-probe
func TestMCPProbe(t *testing.T) {
	// Run all tests in the testdata directory
	mcpscripttest.Test(t, "testdata/*.txt")
}
