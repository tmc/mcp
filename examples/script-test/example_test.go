package scripttest_example

import (
	"testing"

	"github.com/tmc/mcp/mcpscripttest"
)

func TestScripts(t *testing.T) {
	mcpscripttest.TestMinimal(t, "testdata/*.txt")
}
