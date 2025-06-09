package mcpscripttest

import (
	"os"
	"testing"
)

// TestSimpleExample verifies that the stdin_example test script works correctly
func TestSimpleExample(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("Skipping integration test (SKIP_INTEGRATION=1)")
	}
	Test(t, "testdata/scripts/stdin_example.txt")
}
