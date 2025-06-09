// Package serverext provides MCP server management extensions for mcpscripttest.
package serverext

import (
	"rsc.io/script"
)

// DefaultCommands returns the default MCP server management commands for mcpscripttest.
func DefaultCommands() map[string]script.Cmd {
	return map[string]script.Cmd{
		"mcp-server-start":  mcpServerStartCmd,
		"mcp-server-send":   mcpServerSendCmd,
		"mcp-server-stop":   mcpServerStopCmd,
		"mcp-server-output": mcpServerOutputCmd,
		"setenv":            setEnvCmd,
	}
}

// DefaultConditions returns the default MCP server management conditions for mcpscripttest.
func DefaultConditions() map[string]script.Cond {
	return map[string]script.Cond{
		"mcp_server_running":        mcpServerRunningCond,
		"stdio":                     stdioCond,
		"http":                      httpCond,
		"sse":                       sseCond,
		"http_session":              httpSessionCond,
		"multi_connection":          multiConnectionCond,
		"test_server_delay":         testServerDelayCond,
		"test_server_cancel":        testServerCancelCond,
		"test_server_validate_stdout": testServerValidateStdoutCond,
		"server_provided":           serverProvidedCond,
		"server_arg":                serverArgCond,
	}
}

// TODO: These command and condition implementations would need to be moved from the internal package
// For now, we'll create placeholder implementations

var mcpServerStartCmd = script.Command(
	script.CmdUsage{Summary: "start MCP server"},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// TODO: Move implementation from internal/server_commands.go
		return nil, nil
	},
)

var mcpServerSendCmd = script.Command(
	script.CmdUsage{Summary: "send message to MCP server"},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// TODO: Move implementation from internal/server_commands.go
		return nil, nil
	},
)

var mcpServerStopCmd = script.Command(
	script.CmdUsage{Summary: "stop MCP server"},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// TODO: Move implementation from internal/server_commands.go
		return nil, nil
	},
)

var mcpServerOutputCmd = script.Command(
	script.CmdUsage{Summary: "get MCP server output"},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// TODO: Move implementation from internal/server_commands.go
		return nil, nil
	},
)

var setEnvCmd = script.Command(
	script.CmdUsage{Summary: "set environment variable"},
	func(s *script.State, args ...string) (script.WaitFunc, error) {
		// TODO: Move implementation from internal/server_command.go
		return nil, nil
	},
)

// Condition implementations
var mcpServerRunningCond = script.Condition("check if MCP server is running", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var stdioCond = script.Condition("server supports stdio transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return true, nil
})

var httpCond = script.Condition("server supports http transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var sseCond = script.Condition("server supports sse transport", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var httpSessionCond = script.Condition("server supports http sessions", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var multiConnectionCond = script.Condition("server supports multiple connections", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var testServerDelayCond = script.Condition("server supports delay simulation", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var testServerCancelCond = script.Condition("server supports cancellation", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var testServerValidateStdoutCond = script.Condition("server supports stdout validation", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_commands.go
	return false, nil
})

var serverProvidedCond = script.Condition("server command was provided", func(s *script.State) (bool, error) {
	// TODO: Move implementation from internal/server_command.go
	return false, nil
})

var serverArgCond = script.PrefixCondition(
	"check if the server command contains a specific argument",
	func(s *script.State, arg string) (bool, error) {
		// TODO: Move implementation from internal/server_command.go
		return false, nil
	})