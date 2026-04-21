module github.com/tmc/mcp/exp/cmd/mcp2go

go 1.24.3

require (
	github.com/tmc/mcp v0.0.0
	github.com/tmc/mcp/exp v0.0.0
	github.com/tmc/mcp/exp/sourcegen v0.0.0
)

require (
	golang.org/x/tools v0.33.0 // indirect
	rsc.io/script v0.0.2 // indirect
)

replace (
	github.com/tmc/mcp => ../../..
	github.com/tmc/mcp/exp => ../..
	github.com/tmc/mcp/exp/sourcegen => ../../sourcegen
)
