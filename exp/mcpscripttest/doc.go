/*
Package mcpscripttest provides testing utilities for MCP tools using script-based testing.

It wraps the rsc.io/script/scripttest package to provide script-based testing for MCP commands
and servers. The package handles coverage instrumentation, tool installation, and other
MCP-specific test requirements.

# Getting Started

To use mcpscripttest in your project, create a main_test.go file with a test function
that uses the Test or TestWithCoverageOptions function:

	package main

	import (
		"testing"

		"github.com/tmc/mcp/exp/mcpscripttest"
	)

	func TestMyCommand(t *testing.T) {
		// Run script tests with default options
		mcpscripttest.Test(t, "testdata/scripts/*.txt")
	}

# Tool Installation and Coverage

MCP tools are automatically built and installed with coverage instrumentation when GOCOVERDIR
is set in the environment. This feature enables coverage collection for all MCP tools used
in script tests.

The tools package provides fine-grained control:

	func TestWithCustomTools(t *testing.T) {
		toolOpts := &tools.ToolsOptions{
			// Auto-detect coverage from GOCOVERDIR (default: true)
			AutoDetectCoverage: true,

			// Coverage mode: "auto", "on", or "off"
			CoverMode: tools.ToolCoverModeAuto,

			// Custom tool directory
			ToolsDir: "/custom/tools",

			// Select specific tools
			Tools: []string{"mcp-replay", "mcpspy"},

			// Enable verbose logging
			VerboseOutput: true,
		}

		// Install tools with coverage if GOCOVERDIR is set
		cleanup := mcpscripttest.InstallMCPTools(t, toolOpts)
		defer cleanup()

		// Run tests - tools will now collect coverage data
		mcpscripttest.Test(t, "testdata/scripts/*.txt")
	}

# Setting Up Coverage

To enable coverage reporting, set up your main_test.go file with coverage options:

	package main

	import (
		"os"
		"path/filepath"
		"testing"

		"github.com/tmc/mcp/exp/mcpscripttest"
	)

	func TestMyCommand(t *testing.T) {
		// Setup per-test coverage if GOCOVERDIR is set
		var coverageOpts *mcpscripttest.CoverageOptions

		if coverDir := os.Getenv("GOCOVERDIR"); coverDir != "" {
			// Create a specific directory for this test
			myCoverDir := filepath.Join(coverDir, "my-command")
			if err := os.MkdirAll(myCoverDir, 0755); err == nil {
				coverageOpts = &mcpscripttest.CoverageOptions{
					Enabled:              true,
					OutputDir:            myCoverDir,
					PerTestSubdir:        true,
					SaveIntermediateData: true,
					VerboseOutput:        testing.Verbose(),
				}

				t.Logf("Collecting per-test coverage data in %s", myCoverDir)
				t.Logf("To analyze coverage: go tool covdata percent -i %s", myCoverDir)
			}
		}

		// If no coverage options were created, use the defaults
		if coverageOpts == nil {
			coverageOpts = mcpscripttest.DefaultCoverageOptions()
		}

		// Run the scripted tests with our coverage options
		mcpscripttest.TestWithCoverageOptions(t, "testdata/*.txt", coverageOpts)
		mcpscripttest.TestWithCoverageOptions(t, "testdata/scripts/*.txt", coverageOpts)

		// Double-check coverage status at the end
		mcpscripttest.CheckCoverageWarning(t)
	}

# Creating Script Test Files

Create script test files in a testdata/scripts directory within your package.
These files use the txtar format and support all standard scripttest commands
plus MCP-specific commands.

## Example Test File

Here's a simple script test file (testdata/scripts/basic.txt):

	# Test Basic Command Functionality
	mycommand --help
	stdout 'Usage:'
	stderr 'Options:'

	# Test with input file
	>input.txt example content
	exists input.txt
	mycommand process input.txt
	stdout 'Processed'

## MCP-Replay Example

Here's an example test file for MCP-Replay (testdata/mcp-replay-tests.txt):

	# Test MCP-Replay Mock Client/Server Pipeline
	#
	# This test verifies that:
	# 1. mcp-replay runs in various modes (normal, mock-client, mock-server)
	# 2. mcpspy captures communications correctly
	# 3. mcpdiff verifies trace equivalence
	# 4. Server notifications are properly handled

	# === Basic Replay Test ===
	# Verify mcp-replay runs without errors with a basic MCP file
	exec mcp-replay -q -speed=10 basic_sample.mcp

	# === Mock Client/Server Pipeline Test ===
	# Clean up any existing files
	rm -f test-original.mcp
	rm -f test-replay.mcp

	# Create a copy of the test file
	cp basic_sample.mcp test-original.mcp

	# Run the full pipeline
	# Using single quotes to avoid shell interpretation issues
	exec bash -c 'mcp-replay -speed=10 -mock-client test-original.mcp | mcpspy -v -f test-replay.mcp -- mcp-replay -speed=10 -mock-server test-original.mcp'

	# Verify mcpspy created a replay file
	exists test-replay.mcp

	# The captured trace should not be empty
	cat test-replay.mcp
	stdout "jsonrpc"

	# Check that all original messages are present in the new trace
	cat test-replay.mcp
	stdout "initialize"
	stdout "ping"
	stdout "tools/list"
	stdout "tools/call"
	stdout "exit"

	# Test Files

	-- basic_sample.mcp --
	mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
	mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550
	mcp-recv {"jsonrpc":"2.0","method":"ping","id":2} # 1683000001.300
	mcp-send {"jsonrpc":"2.0","id":2,"result":"pong"} # 1683000001.320
	mcp-recv {"jsonrpc":"2.0","method":"tools/list","id":3} # 1683000002.150
	mcp-send {"jsonrpc":"2.0","id":3,"result":{"tools":[{"name":"echo","description":"Echo tool"}]}} # 1683000002.180
	mcp-recv {"jsonrpc":"2.0","method":"tools/call","id":4,"params":{"name":"echo","arguments":{"message":"test"}}} # 1683000003.250
	mcp-send {"jsonrpc":"2.0","id":4,"result":{"message":"test"}} # 1683000003.270
	mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000004.100

## Notification Test Example

Here's an example test for handling notifications (testdata/mcp-replay-notification-test.txt):

	# Test MCP Notification Handling
	#
	# This test verifies that:
	# 1. Normal notifications are processed correctly
	# 2. Auto-notify mode works with mock server

	# === Test Basic Notification Processing ===
	# Run with a file containing notifications
	exec mcp-replay -q -speed=10 notification_test.mcp
	stdout "Error-level message"

	# === Test Auto-Notify Mode ===
	# Clean up any existing files
	rm -f auto-notify-test.mcp

	# Run with auto-notify mode
	exec bash -c 'mcp-replay -mock-client notification_only.mcp | mcpspy -v -f auto-notify-test.mcp -- mcp-replay -mock-server -auto-notify notification_only.mcp'

	# Verify the notifications were processed
	exists auto-notify-test.mcp
	cat auto-notify-test.mcp
	stdout "Test notification"

	# Test Files

	-- notification_test.mcp --
	mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
	mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550
	mcp-send {"method":"notifications/message","params":{"level":"error","data":"Error-level message"},"jsonrpc":"2.0"} # 1683000001.450
	mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000004.100

	-- notification_only.mcp --
	mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000
	mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000000.550
	mcp-send {"method":"notifications/message","params":{"level":"error","data":"Test notification"},"jsonrpc":"2.0"} # 1683000001.450
	mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000004.100

# Directory Structure

A typical project using mcpscripttest should have the following structure:

	mycommand/
	├── main.go           # Command implementation
	├── main_test.go      # Test setup using mcpscripttest
	└── testdata/
	    ├── *.txt         # Root test files
	    └── scripts/
	        └── *.txt     # Script test files

# Coverage and Advanced Options

The package supports coverage collection and advanced options:

	func TestWithCoverage(t *testing.T) {
		// Create custom options
		opts := mcpscripttest.DefaultOptions()
		opts.InstallCoveredTools = true

		// Custom coverage options
		coverOpts := mcpscripttest.DefaultCoverageOptions()

		// Enable per-test subdirectories only if needed, as it makes tests run serially
		// instead of in parallel
		// coverOpts.PerTestSubdir = true

		// Only log coverage details in verbose mode
		coverOpts.VerboseOutput = testing.Verbose()

		// Run tests with custom options
		mcpscripttest.TestWithCoverageOptions(t, "testdata/scripts/*.txt", coverOpts, opts)
	}

# TestMain Support

Use the TestMain function to run deadcode checks only once after all tests are complete:

	func TestMain(m *testing.M) {
		// Use the TestMain helper with default options
		opts := mcpscripttest.DefaultTestMainOptions()

		// Optionally customize options
		opts.RunDeadcodeCheck = true // This is the default
		opts.Cleanup = func() {
			// Any cleanup after all tests
		}

		// Run all tests followed by deadcode check (if enabled)
		code := mcpscripttest.RunTestMain(m, opts)
		os.Exit(code)
	}

# Available MCP Commands

Script tests have access to the following MCP-specific commands installed by default:

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

Additional experimental tools can be installed using DefaultToolsWithScripttestOptions():
- apply-edits: Apply edits from scripttest
- coverage-by-program: Coverage analysis by program
- coverage-hotspots: Find coverage hotspots
- depgraph: Dependency graph generator
- digraph-compat: Digraph compatibility tool
- cmd-docs: Generate command documentation

# Utility Commands

- stdin: Set stdin content from a file for the next command only
- stdout: Verify stdout contains specific text
- setstdin: Set stdin content with more flexible options

Example of using stdin command:

	# Create a sample input file
	>input.json {"jsonrpc":"2.0","method":"initialize","id":1}

	# Use the file as stdin for the next command only
	stdin input.json
	mycommand process  # This command will receive the stdin from input.json
	stdout 'Processed'

	# This command will NOT receive the stdin content, as stdin only affects the next command
	anothercommand  # Normal stdin (empty by default)

Example of using setstdin command:

	# Use previous command's stdout as stdin for the next command
	mycommand generate-data
	setstdin --stdout         # Takes stdout from the previous command
	anothercommand            # Receives the stdout from "mycommand generate-data" as stdin

	# Directly provide text as stdin for the next command
	setstdin {"name": "example", "value": 123}
	mycommand process         # Receives the JSON string as stdin

	# Clear any pending stdin
	setstdin
	mycommand                 # Uses normal stdin (empty by default)

# Custom Commands and Conditions

You can add custom commands and conditions to your script tests:

	func TestWithCustomCommands(t *testing.T) {
		opts := mcpscripttest.DefaultOptions()

		// Add a custom command
		opts.CustomCommands["mycmd"] = script.Command(
			script.CmdUsage{
				Summary: "My custom command",
				Args:    "[args]",
			},
			func(s *script.State, args ...string) (script.WaitFunc, error) {
				// Command implementation
				return nil, nil
			},
		)

		mcpscripttest.Test(t, "testdata/scripts/*.txt", opts)
	}

# Running Tests

To run tests with coverage:

	GOCOVERDIR=/tmp/cover go test -v ./...

To analyze coverage:

	go tool covdata percent -i /tmp/cover/my-command

# See Also

- MCP project: https://github.com/tmc/mcp
- rsc.io/script/scripttest: https://pkg.go.dev/rsc.io/script/scripttest
*/
package mcpscripttest
