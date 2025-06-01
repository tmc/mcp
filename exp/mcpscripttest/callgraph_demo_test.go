package mcpscripttest

import (
	"testing"
)

func TestCallgraphDemonstration(t *testing.T) {
	// Run test demonstrating the static vs dynamic analysis gap
	Test(t, "testdata/callgraph_simple.txt")
}