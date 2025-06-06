package tests

import (
	"github.com/tmc/mcp/exp/mcpscripttest"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest/tools"
)

func TestToolsInstallation(t *testing.T) {
	// Install tools needed for tests
	if testing.Short() {
		t.Skip("Skipping tool installation in short mode")
	}
	
	// Install the tools, including testcallgraph
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"testcallgraph", "mcpdiff"},
	})
	defer cleanup()
}