package main

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestEchoServerBasic(t *testing.T) {
	mcpscripttest.Test(t, "testdata/basic_echo_test.txt")
}

func TestEchoServerTool(t *testing.T) {
	mcpscripttest.Test(t, "testdata/echo_tool_test.txt")
}

func TestEchoServerErrorHandling(t *testing.T) {
	mcpscripttest.Test(t, "testdata/echo_error_handling_test.txt")
}

func TestEchoServerEdgeCases(t *testing.T) {
	mcpscripttest.Test(t, "testdata/echo_edge_cases_test.txt")
}

func TestEchoServerTimestamp(t *testing.T) {
	mcpscripttest.Test(t, "testdata/echo_timestamp_test.txt")
}
