package mcp

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
	"github.com/tmc/mcp/testing/mcpscripttest/tools"
)

// TestMCPWithScripttest runs MCP tests using the scripttest framework
func TestMCPWithScripttest(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	opts := mcpscripttest.DefaultOptions()
	// Run all scripttest files
	mcpscripttest.Test(t, "testdata/scripttest/*.txt", opts)
}

// TestClientServerInteractions tests client-server communication scenarios
func TestClientServerInteractions(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	mcpscripttest.Test(t, "testdata/scripttest/client_server*.txt")
}

// TestTransportMechanisms tests various transport mechanisms
func TestTransportMechanisms(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	mcpscripttest.Test(t, "testdata/scripttest/transport*.txt")
}

// TestProtocolCompliance tests protocol compliance
func TestProtocolCompliance(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	mcpscripttest.Test(t, "testdata/scripttest/protocol*.txt")
}

// TestErrorScenarios tests error handling scenarios
func TestErrorScenarios(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	mcpscripttest.Test(t, "testdata/scripttest/error*.txt")
}

// TestPerformanceScenarios tests performance-related scenarios
func TestPerformanceScenarios(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// Install MCP tools for the tests
	cleanup := mcpscripttest.InstallMCPTools(t, &tools.ToolsOptions{
		Tools: []string{"mcp-probe", "mcpspy", "mcpdiff"},
	})
	defer cleanup()

	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second   // 15 seconds
	opts.TimeoutConfig.DefaultCommandTimeout = 5 * time.Second // 5 seconds
	mcpscripttest.Test(t, "testdata/scripttest/performance*.txt", opts)
}
