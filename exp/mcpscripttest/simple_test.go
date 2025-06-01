package mcpscripttest

import (
	"testing"
)

// TestSkipped skips the test and directs to the experimental repository
func TestSkipped(t *testing.T) {
	t.Skip("Tests for mcpscripttest have been moved to github.com/tmc/mcp-tools-experimental/pkg/mcpscripttest")
}

// Commented out the original test to preserve for reference
/*
// TestBasicStdinFunctionality tests that our approach of setting stdin for commands works
// without relying on the script.State and scripttest framework
func TestBasicStdinFunctionality(t *testing.T) {
	// Original test implementation is available in the experimental repository
}
*/