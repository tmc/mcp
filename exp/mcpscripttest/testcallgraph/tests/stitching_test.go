package tests

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestStitchingNotWorking(t *testing.T) {
	// Test showing that standard callgraph doesn't stitch test scripts to programs
	mcpscripttest.Test(t, "../../testdata/stitching_not_working_simple.txt")
}

func TestStitchingWorking(t *testing.T) {
	// Test showing that our testcallgraph DOES stitch test scripts to programs
	mcpscripttest.Test(t, "../../testdata/stitching_working_simple.txt")
}