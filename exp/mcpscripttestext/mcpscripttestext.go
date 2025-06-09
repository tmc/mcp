// Package mcpscripttestext provides optional command and condition extensions for mcpscripttest.
//
// This package contains the "fancy" commands and conditions that were extracted from the core
// mcpscripttest package to keep the core minimal and focused on essential functionality.
//
// Extensions provided:
//   - bashext: Advanced bash command execution with coverage support
//   - mcptools: MCP tool commands (mcpspy, mcpdiff, etc.)
//   - serverext: MCP server management commands and conditions
//   - conditionsext: Advanced MCP protocol and capability conditions
package mcpscripttestext

import (
	"rsc.io/script"

	"github.com/tmc/mcp/exp/mcpscripttestext/bashext"
	"github.com/tmc/mcp/exp/mcpscripttestext/conditionsext"
	"github.com/tmc/mcp/exp/mcpscripttestext/mcptools"
	"github.com/tmc/mcp/exp/mcpscripttestext/serverext"
)

// DefaultCommands returns all the default extension commands combined.
func DefaultCommands() map[string]script.Cmd {
	commands := make(map[string]script.Cmd)
	
	// Add all extension commands
	for name, cmd := range bashext.DefaultCommands() {
		commands[name] = cmd
	}
	for name, cmd := range mcptools.DefaultCommands() {
		commands[name] = cmd
	}
	for name, cmd := range serverext.DefaultCommands() {
		commands[name] = cmd
	}
	for name, cmd := range conditionsext.DefaultCommands() {
		commands[name] = cmd
	}
	
	return commands
}

// DefaultConditions returns all the default extension conditions combined.
func DefaultConditions() map[string]script.Cond {
	conditions := make(map[string]script.Cond)
	
	// Add all extension conditions
	for name, cond := range bashext.DefaultConditions() {
		conditions[name] = cond
	}
	for name, cond := range mcptools.DefaultConditions() {
		conditions[name] = cond
	}
	for name, cond := range serverext.DefaultConditions() {
		conditions[name] = cond
	}
	for name, cond := range conditionsext.DefaultConditions() {
		conditions[name] = cond
	}
	
	return conditions
}

// BashCommands returns only the bash extension commands.
func BashCommands() map[string]script.Cmd {
	return bashext.DefaultCommands()
}

// BashConditions returns only the bash extension conditions.
func BashConditions() map[string]script.Cond {
	return bashext.DefaultConditions()
}

// MCPToolCommands returns only the MCP tool commands.
func MCPToolCommands() map[string]script.Cmd {
	return mcptools.DefaultCommands()
}

// MCPToolConditions returns only the MCP tool conditions.
func MCPToolConditions() map[string]script.Cond {
	return mcptools.DefaultConditions()
}

// ServerCommands returns only the server management commands.
func ServerCommands() map[string]script.Cmd {
	return serverext.DefaultCommands()
}

// ServerConditions returns only the server management conditions.
func ServerConditions() map[string]script.Cond {
	return serverext.DefaultConditions()
}

// AdvancedConditions returns only the advanced protocol conditions.
func AdvancedConditions() map[string]script.Cond {
	return conditionsext.DefaultConditions()
}

// ApplyToOptions is a helper function to apply all extensions to mcpscripttest options.
func ApplyToOptions(opts interface{}) {
	// TODO: This would need to be implemented once we update the core mcpscripttest
	// to use these extensions. The function would merge the extension commands and
	// conditions into the provided options structure.
}