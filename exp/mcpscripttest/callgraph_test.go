package mcpscripttest

import (
	"testing"
)

func TestCallgraphLimitation(t *testing.T) {
	// Run tests demonstrating the static vs dynamic analysis gap
	Test(t, "testdata/execution_gap_inline.txt")
}

func TestSimpleCallgraphDemo(t *testing.T) {
	// Simple demo showing the conceptual limitation
	Test(t, "testdata/callgraph_simple_demo.txt")
}