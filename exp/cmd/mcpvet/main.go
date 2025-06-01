// Command mcpvet is a tool for analyzing Go packages and scripttest files.
//
// mcpvet is intended to be used with the "go vet" command, to run as part of
// standard Go development and CI workflows.
//
// Usage:
//
//	mcpvet [-flag] [package]
//	go vet -vettool=`which mcpvet` [-flag] [package]
//
// For more information, see:
//
//	go doc golang.org/x/tools/go/analysis
package main

import (
	"github.com/tmc/mcp/exp/cmd/mcpvet/scripttest"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	// Run the analyzer using the singlechecker engine, which handles the flags.
	singlechecker.Main(scripttest.Analyzer)
}
