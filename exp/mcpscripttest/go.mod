module github.com/tmc/mcp/exp/mcpscripttest

go 1.23.0

toolchain go1.24.3

require (
	golang.org/x/tools v0.33.0
	rsc.io/script v0.0.2
)

require (
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
)

replace github.com/tmc/mcp => ../..
