package mcp

import (
	"testing"
	"time"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestMCPComprehensive runs comprehensive scripttest tests for the MCP package
func TestMCPComprehensive(t *testing.T) {
	// Disable overall timeout to debug the issue
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 0 // Disable overall timeout for debugging
	mcpscripttest.Test(t, "testdata/scripttest/*.txt", opts)
}

// TestMCPClientServer runs client-server interaction tests
func TestMCPClientServer(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripttest/client_server*.txt")
}

// TestMCPTransports runs transport layer tests
func TestMCPTransports(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripttest/transport*.txt")
}

// TestMCPProtocol runs protocol-level tests
func TestMCPProtocol(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripttest/protocol*.txt")
}

// TestMCPErrorHandling runs error handling and edge case tests
func TestMCPErrorHandling(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripttest/error*.txt")
}

// TestMCPPerformance runs performance and timeout tests
func TestMCPPerformance(t *testing.T) {
	// Use aggressive timeout configuration
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second   // 15 seconds
	opts.TimeoutConfig.DefaultCommandTimeout = 5 * time.Second // 5 seconds
	mcpscripttest.Test(t, "testdata/scripttest/performance*.txt", opts)
}
