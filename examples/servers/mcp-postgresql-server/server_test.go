package main

import (
	"os"
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestMain(m *testing.M) {
	code := mcpscripttest.RunWithCoverage(m, "postgresql-server")
	os.Exit(code)
}

func TestPostgreSQLServer(t *testing.T) {
	mcpscripttest.RunDirectoryTests(t, "testdata")
}
