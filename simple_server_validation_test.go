package mcp

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

// TestSimpleServerValidation tests basic server validation
func TestSimpleServerValidation(t *testing.T) {
	opts := mcpscripttest.DefaultOptions()

	mcpscripttest.Test(t, "testdata/scripttest/simple_server_validation_test.txt", opts)
}
