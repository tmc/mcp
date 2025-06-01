package mcp

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestSimpleServerValidation(t *testing.T) {
	// Test simple server builds with default timeouts
	opts := mcpscripttest.DefaultOptions()
	opts.TimeoutConfig.TestOverallTimeout = 0 // Remove overall timeout
	mcpscripttest.Test(t, "testdata/scripttest/simple_server_validation_test.txt", opts)
}
