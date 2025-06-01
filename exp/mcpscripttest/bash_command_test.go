package mcpscripttest

import (
	"testing"
)

func TestBashCommand(t *testing.T) {
	// Run the bash command test script
	Test(t, "testdata/bash_command_test.txt", DefaultOptions())
}