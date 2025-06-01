package mcpscripttest

import (
	"testing"
)

func TestToolsInstallation(t *testing.T) {
	// Install tools needed for tests
	if testing.Short() {
		t.Skip("Skipping tool installation in short mode")
	}
	
	// Install the tools, including testcallgraph
	cleanup := InstallMCPTools(t, &ToolsOptions{
		Tools: []string{"testcallgraph", "mcpdiff"},
	})
	defer cleanup()
}