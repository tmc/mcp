package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMcp2goScripts(t *testing.T) {
	// Run all script tests in testdata/
	mcpscripttest.Test(t, "testdata/*.txt", nil)
}