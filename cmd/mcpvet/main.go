// Mcpvet checks Go packages for incorrect MCP SDK registrations.
// Build it and run go vet -vettool=/path/to/mcpvet ./... .
package main

import (
	"github.com/tmc/mcp/mcpanalysis/toolinputschema"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(toolinputschema.Analyzer) }
