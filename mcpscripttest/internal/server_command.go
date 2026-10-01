package internal

import (
	"fmt"
	"os"
	"strings"

	"rsc.io/script"
)

// addServerCommandSupport adds support for using the provided server command
// in test scripts by exposing it as variables and conditions
func addServerCommandSupport(e *script.Engine) {
	// Get the server command from the environment
	serverCmd := os.Getenv("MCP_SERVER_COMMAND")
	if serverCmd == "" {
		serverCmd = "echo \"No server command provided\""
	}

	// Add the server command and HTTP port to the environment
	// These will be accessible in test scripts via $MCP_SERVER_COMMAND and $MCP_HTTP_PORT

	// Add HTTP port from environment
	httpPort := os.Getenv("MCP_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8765" // Default port
	}

	// Add a helper condition to check if a server command was provided
	e.Conds["server_provided"] = script.Condition("server command was provided", func(s *script.State) (bool, error) {
		sc := os.Getenv("MCP_SERVER_COMMAND")
		return sc != "" && sc != "echo \"No server command provided\"", nil
	})

	// Add helper condition to check if server command contains a specific argument
	e.Conds["server_arg"] = script.PrefixCondition(
		"check if the server command contains a specific argument",
		func(s *script.State, arg string) (bool, error) {
			if arg == "" {
				return false, fmt.Errorf("missing argument")
			}
			sc := os.Getenv("MCP_SERVER_COMMAND")
			return strings.Contains(sc, arg), nil
		})

	// Add custom command to set environment variables at runtime
	e.Cmds["setenv"] = script.Command(
		script.CmdUsage{
			Summary: "set environment variable",
			Args:    "name value",
		},
		func(s *script.State, args ...string) (script.WaitFunc, error) {
			if len(args) != 2 {
				return nil, script.ErrUsage
			}
			return nil, s.Setenv(args[0], args[1])
		})
}
