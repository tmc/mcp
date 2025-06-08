package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestSQLiteServer(t *testing.T) {
	// Use aggressive timeout configuration for comprehensive testing
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second   // Aggressive: 15s total
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second // Aggressive: 3s per command
	opts.TimeoutConfig.ServerStartupTimeout = 2 * time.Second  // Aggressive: 2s startup
	opts.TimeoutConfig.ServerResponseTimeout = 1 * time.Second // Aggressive: 1s response
	mcpscripttest.Test(t, "testdata/*.txt", opts)
}

func TestSQLiteServerBasic(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 10 * time.Second   // Aggressive: 10s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second // Aggressive: 2s per command
	mcpscripttest.Test(t, "testdata/basic_sqlite_test.txt", opts)
}

func TestSQLiteServerAdvanced(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 20 * time.Second   // Aggressive: 20s total
	opts.TimeoutConfig.DefaultCommandTimeout = 4 * time.Second // Aggressive: 4s per command
	mcpscripttest.Test(t, "testdata/sqlite_advanced_test.txt", opts)
}

func TestSQLiteServerErrors(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 8 * time.Second    // Aggressive: 8s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second // Aggressive: 2s per command
	mcpscripttest.Test(t, "testdata/sqlite_error_handling_test.txt", opts)
}

func TestSQLiteServerPerformance(t *testing.T) {
	// Ultra-aggressive timeout configuration for performance tests
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 12 * time.Second          // Ultra-aggressive: 12s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second        // Ultra-aggressive: 2s per command
	opts.TimeoutConfig.ServerStartupTimeout = 1 * time.Second         // Ultra-aggressive: 1s startup
	opts.TimeoutConfig.ServerResponseTimeout = 500 * time.Millisecond // Ultra-aggressive: 500ms response
	mcpscripttest.Test(t, "testdata/sqlite_performance_test.txt", opts)
}
