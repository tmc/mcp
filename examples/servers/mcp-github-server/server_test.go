package main

import (
	"os"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMain(m *testing.M) {
	code := mcpscripttest.RunWithCoverage(m, "github-server")
	os.Exit(code)
}

func TestGitHubServer(t *testing.T) {
	mcpscripttest.RunDirectoryTests(t, "testdata")
}
