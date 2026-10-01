package toolinputschema_test

import (
	"testing"

	"github.com/tmc/mcp/mcpanalysis/toolinputschema"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), toolinputschema.Analyzer, "a")
}
