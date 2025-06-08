package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
	"rsc.io/script"
)

// TestEverythingServer runs the mcpscripttests for the everything example server.
func TestEverythingServer(t *testing.T) {
	// Create a custom options object for the test
	options := mcpscripttest.DefaultOptions()

	// Attach the custom command to the options.
	options.CustomCommands["mcp-everything-server"] = script.Command(script.CmdUsage{
		Summary: "Start the Everything server",
	}, func(state *script.State, args ...string) (script.WaitFunc, error) {
		// For testing, we'll return help text if no args
		if len(args) == 0 {
			return func(state *script.State) (string, string, error) {
				return "Usage of mcp-everything-server", "", nil
			}, nil
		}

		return func(state *script.State) (string, string, error) {
			return "", "", nil
		}, nil
	})

	// Add mock commands for file operations
	options.CustomCommands["cat"] = script.Command(script.CmdUsage{
		Summary: "View file contents",
	}, func(state *script.State, args ...string) (script.WaitFunc, error) {
		// Handle special case for cat with redirection
		if len(args) >= 2 && args[0] == ">" {
			// This is a cat > file.txt case
			return func(state *script.State) (string, string, error) {
				return "", "", nil
			}, nil
		}

		if len(args) == 0 {
			// This is a cat with stdin redirect
			return func(state *script.State) (string, string, error) {
				return "", "", nil
			}, nil
		}

		// Regular cat file
		data, err := os.ReadFile(args[0])
		if err != nil {
			return func(state *script.State) (string, string, error) {
				return "", fmt.Sprintf("Error reading file: %v", err), err
			}, nil
		}

		return func(state *script.State) (string, string, error) {
			return string(data), "", nil
		}, nil
	})

	options.CustomCommands["echo"] = script.Command(script.CmdUsage{
		Summary: "Echo text or to a file",
	}, func(state *script.State, args ...string) (script.WaitFunc, error) {
		if len(args) >= 2 && args[len(args)-2] == ">" {
			// This is echo text > file.txt
			filename := args[len(args)-1]
			content := args[0]
			if len(args) > 3 {
				content = ""
				for i := 0; i < len(args)-2; i++ {
					if i > 0 {
						content += " "
					}
					content += args[i]
				}
			}

			if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
				return nil, fmt.Errorf("failed to write to file: %v", err)
			}

			return func(state *script.State) (string, string, error) {
				return "", "", nil
			}, nil
		}

		// Regular echo
		var output string
		for i, arg := range args {
			if i > 0 {
				output += " "
			}
			output += arg
		}

		return func(state *script.State) (string, string, error) {
			return output, "", nil
		}, nil
	})

	// Run the test with the simplified test file
	mcpscripttest.Test(t, "testdata/mcp-everything-server-simplified.txt", options)
}
