package main

import (
	"testing"

	"github.com/tmc/mcp/testing/mcpscripttest"
)

func TestTimeServerBasic(t *testing.T) {
	mcpscripttest.Test(t, "testdata/basic_time_test.txt")
}

func TestTimeServerTools(t *testing.T) {
	mcpscripttest.Test(t, "testdata/time_tools_test.txt")
}

func TestTimeServerConversion(t *testing.T) {
	mcpscripttest.Test(t, "testdata/time_conversion_test.txt")
}

func TestTimeServerErrorHandling(t *testing.T) {
	mcpscripttest.Test(t, "testdata/time_error_handling_test.txt")
}

func TestTimeServerTimezones(t *testing.T) {
	mcpscripttest.Test(t, "testdata/timezone_validation_test.txt")
}

func TestTimeServerDST(t *testing.T) {
	mcpscripttest.Test(t, "testdata/dst_handling_test.txt")
}
