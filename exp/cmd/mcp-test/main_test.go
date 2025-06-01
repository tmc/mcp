package main

import (
	"testing"
)

func TestSkipped(t *testing.T) {
	t.Skip("Tests for mcp-test have been moved to github.com/tmc/mcp-tools-experimental/cmd/mcp-test")
}