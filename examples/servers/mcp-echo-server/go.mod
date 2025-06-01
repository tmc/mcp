module github.com/tmc/mcp/examples/servers/mcp-echo-server

go 1.23

require (
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
	github.com/tmc/mcp/exp/mcpscripttest v0.0.0-00010101000000-000000000000
)

replace github.com/tmc/mcp => ../../../

replace github.com/tmc/mcp/exp/mcpscripttest => ../../../exp/mcpscripttest
