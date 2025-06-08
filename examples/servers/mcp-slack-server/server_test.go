package main

import (
	"testing"
	"time"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestSlackServer(t *testing.T) {
	// Use aggressive timeout configuration for performance testing
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 15 * time.Second
	opts.TimeoutConfig.DefaultCommandTimeout = 3 * time.Second
	opts.TimeoutConfig.ServerStartupTimeout = 2 * time.Second
	opts.TimeoutConfig.ServerResponseTimeout = 1 * time.Second
	mcpscripttest.Test(t, "testdata/*.txt", opts)
}
