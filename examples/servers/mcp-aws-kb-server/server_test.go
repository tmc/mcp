package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestAWSKBServerBasic(t *testing.T) {
	mcpscripttest.RunTest(t, "testdata/basic_aws_kb_test.txt")
}

func TestAWSKBServerTools(t *testing.T) {
	mcpscripttest.RunTest(t, "testdata/aws_kb_tools_test.txt")
}

func TestAWSKBServerErrorHandling(t *testing.T) {
	mcpscripttest.RunTest(t, "testdata/aws_kb_error_handling_test.txt")
}

func TestAWSKBServerPerformance(t *testing.T) {
	mcpscripttest.RunTest(t, "testdata/aws_kb_performance_test.txt")
}
