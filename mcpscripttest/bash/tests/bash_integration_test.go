package tests

import (
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
)

func TestBashCallGraphIntegration(t *testing.T) {
	mcpscripttest.Test(t, "../../testdata/testcallgraph_bash_test.txt")
}
