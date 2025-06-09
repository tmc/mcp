module github.com/tmc/mcp/testing/mcpscripttest/fuzzing

go 1.23.0

toolchain go1.24.3

require github.com/tmc/mcp/testing/mcpscripttest v0.0.0-00010101000000-000000000000

require (
	golang.org/x/tools v0.33.0 // indirect
	rsc.io/script v0.0.2 // indirect
)

replace github.com/tmc/mcp/testing/mcpscripttest => ../
