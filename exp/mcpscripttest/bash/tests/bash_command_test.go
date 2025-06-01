package tests

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestBashCommand(t *testing.T) {
	// Run the bash command test script
	mcpscripttest.Test(t, "../../testdata/bash_command_test.txt", mcpscripttest.DefaultOptions())
}