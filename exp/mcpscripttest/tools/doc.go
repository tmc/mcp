/*
Package tools provides tool installation and management for MCP testing.

This package handles the installation, building, and management of MCP tools
used in testing scenarios, including coverage instrumentation.

# Tool Installation

The package provides functions to install MCP tools with optional coverage instrumentation:

	import (
		"testing"
		"github.com/tmc/mcp/exp/mcpscripttest/tools"
	)

	func TestWithTools(t *testing.T) {
		// Install default MCP tools
		cleanup := tools.InstallMCPTools(t, nil)
		defer cleanup()
		
		// Tools are now available in PATH
		// Run your tests here
	}

# Coverage Instrumentation

When GOCOVERDIR is set, tools are automatically built with coverage instrumentation:

	func TestWithCoverage(t *testing.T) {
		opts := &tools.ToolsOptions{
			CoverMode: tools.ToolCoverModeAuto, // Auto-detect from GOCOVERDIR
		}
		
		cleanup := tools.InstallMCPTools(t, opts)
		defer cleanup()
		
		// Tools will collect coverage data when GOCOVERDIR is set
	}

# Available Tools

Default tools installed by DefaultToolsOptions():
- mcp-replay: Replay MCP recordings
- mcpspy: Monitor MCP communications
- mcp-shadow: Shadow traffic for testing
- mcp-send: Send MCP messages
- mcpdiff: Compare MCP traces
- mcp-probe: Probe MCP servers
- mcpcat: Display MCP traces with color
- mcp-sort: Sort MCP traces
- mcp-connect: Connect to MCP servers
- mcp-proxy: Protocol proxy for debugging
- mcp-serve: Serve MCP endpoints
- mcp-debug: Debug MCP servers

Additional tools available with DefaultToolsWithScripttestOptions():
- apply-edits: Apply edits from scripttest
- coverage-by-program: Coverage analysis by program
- coverage-hotspots: Find coverage hotspots
- depgraph: Dependency graph generator
- digraph-compat: Digraph compatibility tool
- cmd-docs: Generate command documentation

# Custom Tool Selection

You can specify which tools to install:

	opts := &tools.ToolsOptions{
		Tools: []string{"mcpdiff", "mcpspy", "mcp-replay"},
		VerboseOutput: true,
	}
	
	cleanup := tools.InstallMCPTools(t, opts)
	defer cleanup()

# Tool Directory Management

By default, tools are installed in a temporary directory that is cleaned up
after tests. You can specify a custom directory:

	opts := &tools.ToolsOptions{
		ToolsDir: "/path/to/tools",
	}
	
	cleanup := tools.InstallMCPTools(t, opts)
	defer cleanup() // Won't delete custom directory

# Coverage Modes

The package supports three coverage modes:

- ToolCoverModeAuto: Automatically detect from GOCOVERDIR (default)
- ToolCoverModeOn: Always build with coverage instrumentation
- ToolCoverModeOff: Never build with coverage instrumentation

# PATH Management

The package handles PATH manipulation automatically:

1. Original PATH is saved before modification
2. Tool directory is prepended to PATH
3. Cleanup function restores original PATH

For existing tool directories, use SetupMCPToolsPath:

	// Just add existing tools to PATH without installing
	cleanup := tools.SetupMCPToolsPath(t, "/existing/tools")
	defer cleanup()
*/
package tools
