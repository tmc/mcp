package mcpscripttest

import (
	"testing"
)

func TestStitchingNotWorking(t *testing.T) {
	// Test showing that standard callgraph doesn't stitch test scripts to programs
	Test(t, "testdata/stitching_not_working_simple.txt")
}

func TestStitchingWorking(t *testing.T) {
	// Test showing that our testcallgraph DOES stitch test scripts to programs
	Test(t, "testdata/stitching_working_simple.txt")
}