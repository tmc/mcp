//go:build standalone
// +build standalone

package main

import (
	"github.com/tmc/mcp/exp/mcpscripttest"
)

// standaloneMain provides an alternative entry point that doesn't use the testing package
func standaloneMain() int {
	// Parse command-line flags (reuse the same flag logic from main.go)
	// ... flag parsing logic ...

	// Create a custom runner that avoids testing.Short()
	runner := &mcpscripttest.TestRunner{
		Options: &mcpscripttest.MCPScripttestOptions{
			IncludeDefaultMCPCommands: true,
			AdditionalEnvVars:         map[string]string{},
		},
		Verbose: false,
	}

	// Set up pattern
	pattern := "testdata/mcp_conformance/*.txt"

	// Run tests without the testing framework
	return runner.RunTests(pattern)
}

// Build with: go build -tags standalone
