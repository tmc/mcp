package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
	"rsc.io/script"
)

// Pre-find the claude executable before any environment changes
var realClaudePath, _ = exec.LookPath("claude")

// Pre-find node executable to check dependencies
var nodeAvailable = false

func init() {
	nodePath, err := exec.LookPath("node")
	if err == nil && nodePath != "" {
		nodeAvailable = true
	}
}

func TestMain(m *testing.M) {
	// Run the tests without custom setup
	// mcpscripttest will handle command setup
	os.Exit(m.Run())
}

func TestMCPDiff(t *testing.T) {
	// Run scripttest files without custom claude command
	// The framework will handle tool installation and setup
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}

// mockClaudeCommand provides a mock implementation of Claude for testing
func mockClaudeCommand(args []string) (script.WaitFunc, error) {
	// For version flag, return mock version
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
		return func(s *script.State) (string, string, error) {
			return "claude version 1.0.0 (mock for testing)\n", "", nil
		}, nil
	}

	// For capabilities, return mock response
	if len(args) > 0 && args[0] == "capabilities" {
		return func(s *script.State) (string, string, error) {
			return `{"models":["claude-3-opus-20240229","claude-3-sonnet-20240229"],"tools":["bash","read_file"]}`, "", nil
		}, nil
	}

	// Special handling for MCP commands
	if len(args) > 0 && args[0] == "mcp" {
		if len(args) > 1 && args[1] == "list" {
			return func(s *script.State) (string, string, error) {
				return "MOCK - Claude MCP list command\nAvailable MCP servers:\n  - mock-time-server\n  - mock-file-server\n", "", nil
			}, nil
		}

		// Generic MCP command response
		return func(s *script.State) (string, string, error) {
			return fmt.Sprintf("MOCK - Claude MCP command: %s\n", strings.Join(args[1:], " ")), "", nil
		}, nil
	}

	// For other prompts, return a mock response
	prompt := strings.Join(args, " ")
	return func(s *script.State) (string, string, error) {
		return fmt.Sprintf("MOCK - Claude would have processed: %s\n", prompt), "", nil
	}, nil
}
