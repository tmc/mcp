package tests

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"

	"github.com/tmc/mcp/testing/mcpscripttest/tools"
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
