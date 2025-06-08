package main

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestMCPReplay(t *testing.T) {
	mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
