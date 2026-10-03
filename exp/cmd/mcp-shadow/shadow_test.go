package main

import (
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
	"github.com/tmc/mcp/mcpscripttest/tools"
)

// TestMCPShadowScripts runs all scripts in the testdata directory
func TestMCPShadowScripts(t *testing.T) {
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{Tools: []string{"mcp-shadow"}})
	defer cleanup()
	mcpscripttest.Test(t, "testdata/*.txt")
}
