package main

import (
	"testing"
)

// TestSkipped skips the test and directs to the experimental repository
func TestSkipped(t *testing.T) {
	t.Skip("Tests for mcptrace2gostruct have been moved to github.com/tmc/mcp-tools-experimental/cmd/mcptrace2gostruct")
}

// Commented test functions follow to preserve code for reference
/*
// Original test implementations are available in the experimental repository
*/