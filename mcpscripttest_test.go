package mcp_test

import (
	"path/filepath"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPScriptTest runs the mcpscripttest tests in testdata directories
func TestMCPScriptTest(t *testing.T) {
	// Run tests in testdata/server_coverage
	t.Run("server_coverage", func(t *testing.T) {
		// Test each script file separately for better error reporting
		files := []string{
			"basic_server.txt",
			"error_handling.txt",
			"sse_transport.txt",
		}

		for _, file := range files {
			t.Run(file, func(t *testing.T) {
				testPath := filepath.Join("testdata", "server_coverage", file)
				mcpscripttest.Test(t, testPath)
			})
		}
	})

	// Run tests in testdata/tools if they are scripttest compatible
	t.Run("tools", func(t *testing.T) {
		testPath := filepath.Join("testdata", "tools", "integration.txt")
		mcpscripttest.Test(t, testPath)
	})
}

// TestServerCoverageWithOptions runs server coverage tests with custom options
func TestServerCoverageWithOptions(t *testing.T) {
	// Setup custom options
	opts := mcpscripttest.DefaultOptions()

	// Enable debug mode if testing with -v
	if testing.Verbose() {
		opts.DebugMode = true
	}

	// Test each file separately
	files := []string{
		"basic_server.txt",
		"error_handling.txt",
		"sse_transport.txt",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			testPath := filepath.Join("testdata", "server_coverage", file)
			mcpscripttest.Test(t, testPath, opts)
		})
	}
}
