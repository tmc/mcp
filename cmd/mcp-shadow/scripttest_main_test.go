package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPShadow runs all script tests for mcp-shadow
func TestMCPShadow(t *testing.T) {
	// Run all tests in the testdata directory
	mcpscripttest.Test(t, "testdata/*.txt")
}
