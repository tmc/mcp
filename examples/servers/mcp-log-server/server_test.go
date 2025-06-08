package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// TestLogServerBasic runs basic log server tests
func TestLogServerBasic(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second
	mcpscripttest.Test(t, "testdata/basic_log_test.txt", opts)
}

// TestLogServerPerformance runs performance tests with aggressive timeouts
func TestLogServerPerformance(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig = mcpscripttest.UltraAggressiveTimeoutConfig()
	mcpscripttest.Test(t, "testdata/log_performance_test.txt", opts)
}
