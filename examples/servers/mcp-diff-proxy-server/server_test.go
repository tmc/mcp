package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestDiffProxyServer(t *testing.T) {
	// Use aggressive timeout configuration for comprehensive testing
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 12 * time.Second   // Aggressive: 12s total
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second // Aggressive: 3s per command
	opts.TimeoutConfig.ServerStartupTimeout = 2 * time.Second  // Aggressive: 2s startup
	opts.TimeoutConfig.ServerResponseTimeout = 1 * time.Second // Aggressive: 1s response
	mcpscripttest.Test(t, "testdata/*.txt", opts)
}

func TestDiffProxyServerBasic(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 8 * time.Second    // Aggressive: 8s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second // Aggressive: 2s per command
	mcpscripttest.Test(t, "testdata/basic_diff_proxy_test.txt", opts)
}

func TestDiffProxyServerErrors(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 6 * time.Second            // Aggressive: 6s total
	opts.TimeoutConfig.DefaultCommandTimeout = 1500 * time.Millisecond // Aggressive: 1.5s per command
	mcpscripttest.Test(t, "testdata/diff_proxy_error_handling_test.txt", opts)
}

func TestDiffProxyServerPerformance(t *testing.T) {
	// Ultra-aggressive timeout configuration for performance tests
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 10 * time.Second          // Ultra-aggressive: 10s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second        // Ultra-aggressive: 2s per command
	opts.TimeoutConfig.ServerStartupTimeout = 1 * time.Second         // Ultra-aggressive: 1s startup
	opts.TimeoutConfig.ServerResponseTimeout = 500 * time.Millisecond // Ultra-aggressive: 500ms response
	mcpscripttest.Test(t, "testdata/diff_proxy_performance_test.txt", opts)
}
