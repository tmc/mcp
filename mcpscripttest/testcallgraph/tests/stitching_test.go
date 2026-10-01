package tests

import (
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
	"github.com/tmc/mcp/mcpscripttest/tools"
)

func TestStitchingNotWorking(t *testing.T) {
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		CoverMode: tools.ToolCoverModeOff,
		Tools:     []string{"mcpdiff"},
	})
	defer cleanup()

	// Test showing that standard callgraph doesn't stitch test scripts to programs
	mcpscripttest.Test(t, "../../testdata/stitching_not_working_simple.txt")
}

func TestStitchingWorking(t *testing.T) {
	// Test showing that our testcallgraph DOES stitch test scripts to programs
	mcpscripttest.Test(t, "../../testdata/stitching_working_simple.txt")
}
