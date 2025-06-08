package main

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestMCPSpy(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
