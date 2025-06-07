package main

import (
	"testing"

	"github.com/tmc/mcp/exp/mcpscripttest"
)

func TestGitLabServer(t *testing.T) {
	mcpscripttest.Test(t, "testdata/basic_gitlab_test.txt")
}

func TestGitLabServerTools(t *testing.T) {
	mcpscripttest.Test(t, "testdata/gitlab_tools_test.txt")
}

func TestGitLabServerErrorHandling(t *testing.T) {
	mcpscripttest.Test(t, "testdata/gitlab_error_handling_test.txt")
}
