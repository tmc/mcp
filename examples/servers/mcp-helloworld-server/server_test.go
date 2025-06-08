package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestHelloWorldServer(t *testing.T) {
	// Use aggressive timeout configuration for comprehensive testing
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 12 * time.Second          // Aggressive: 12s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second        // Aggressive: 2s per command
	opts.TimeoutConfig.ServerStartupTimeout = 1 * time.Second         // Aggressive: 1s startup
	opts.TimeoutConfig.ServerResponseTimeout = 800 * time.Millisecond // Aggressive: 800ms response
	mcpscripttest.Test(t, "testdata/*.txt", opts)
}

func TestHelloWorldServerBasic(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 8 * time.Second            // Aggressive: 8s total
	opts.TimeoutConfig.DefaultCommandTimeout = 1500 * time.Millisecond // Aggressive: 1.5s per command
	mcpscripttest.Test(t, "testdata/basic_helloworld_test.txt", opts)
}

func TestHelloWorldServerLanguages(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second   // Aggressive: 15s total
	opts.TimeoutConfig.DefaultCommandTimeout = 2 * time.Second // Aggressive: 2s per command
	mcpscripttest.Test(t, "testdata/helloworld_languages_test.txt", opts)
}

func TestHelloWorldServerErrors(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 6 * time.Second    // Aggressive: 6s total
	opts.TimeoutConfig.DefaultCommandTimeout = 1 * time.Second // Aggressive: 1s per command
	mcpscripttest.Test(t, "testdata/helloworld_error_handling_test.txt", opts)
}

func TestHelloWorldServerPerformance(t *testing.T) {
	// Ultra-aggressive timeout configuration for performance tests
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 8 * time.Second           // Ultra-aggressive: 8s total
	opts.TimeoutConfig.DefaultCommandTimeout = 1 * time.Second        // Ultra-aggressive: 1s per command
	opts.TimeoutConfig.ServerStartupTimeout = 500 * time.Millisecond  // Ultra-aggressive: 500ms startup
	opts.TimeoutConfig.ServerResponseTimeout = 400 * time.Millisecond // Ultra-aggressive: 400ms response
	mcpscripttest.Test(t, "testdata/helloworld_performance_test.txt", opts)
}
