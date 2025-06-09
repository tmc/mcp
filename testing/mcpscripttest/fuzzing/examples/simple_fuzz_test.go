package examples

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest/fuzzing"
)

// TestSimpleFuzzing demonstrates basic fuzzing
func TestSimpleFuzzing(t *testing.T) {
	t.Skip("Example test")
}

// FuzzScriptTest is the actual fuzz function
func FuzzScriptTest(f *testing.F) {
	// Use the fuzzing package directly
	fuzzing.FuzzScriptTest(f)
}
