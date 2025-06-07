package main

import (
	"os"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMain(m *testing.M) {
	code := mcpscripttest.RunWithCoverage(m, "bravesearch-server")
	os.Exit(code)
}

func TestBraveSearchServer(t *testing.T) {
	mcpscripttest.RunDirectoryTests(t, "testdata")
}
