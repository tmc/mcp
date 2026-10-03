package main

import (
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
	"github.com/tmc/mcp/mcpscripttest/tools"
)

func TestMCPReplay(t *testing.T) {
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{Tools: []string{"mcp-replay"}})
	defer cleanup()
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
