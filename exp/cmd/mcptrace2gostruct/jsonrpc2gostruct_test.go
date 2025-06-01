// Package main implements tests for the mcptrace2gostruct tool
package main

import (
	"testing"
)

// TestJSONRPCSkipped skips the test and directs to the experimental repository
func TestJSONRPCSkipped(t *testing.T) {
	t.Skip("Tests for mcptrace2gostruct have been moved to github.com/tmc/mcp-tools-experimental/cmd/mcptrace2gostruct")
}
