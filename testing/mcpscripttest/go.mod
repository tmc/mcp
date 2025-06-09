module github.com/tmc/mcp/testing/mcpscripttest

go 1.23.0

toolchain go1.24.3

require rsc.io/script v0.0.2

require golang.org/x/tools v0.33.0

require github.com/tmc/mcp v0.0.0-20241126155658-6dc8f6842a0a // indirect

replace github.com/tmc/mcp => ../..

tool (
	github.com/tmc/mcp/cmd/mcp-shadow
	github.com/tmc/mcp/cmd/mcpdiff
)
