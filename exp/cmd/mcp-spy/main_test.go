package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMCPSpy(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
