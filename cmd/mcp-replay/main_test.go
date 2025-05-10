package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMCPReplay(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
