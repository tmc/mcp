package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

// TestJSONServerBasic runs basic JSON server tests
func TestJSONServerBasic(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second
	mcpscripttest.Test(t, "testdata/basic_json_test.txt", opts)
}

// TestJSONServerTools runs JSON server tools tests
func TestJSONServerTools(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 20 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 4 * time.Second
	mcpscripttest.Test(t, "testdata/json_tools_test.txt", opts)
}

// TestJSONServerValidation runs JSON validation tests
func TestJSONServerValidation(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 12 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second
	mcpscripttest.Test(t, "testdata/json_validation_test.txt", opts)
}

// TestJSONServerEdgeCases runs edge case tests
func TestJSONServerEdgeCases(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 18 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 4 * time.Second
	mcpscripttest.Test(t, "testdata/json_edge_cases_test.txt", opts)
}

// TestJSONServerPerformance runs performance tests with aggressive timeouts
func TestJSONServerPerformance(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig = mcpscripttest.UltraAggressiveTimeoutConfig()
	mcpscripttest.Test(t, "testdata/json_performance_test.txt", opts)
}
