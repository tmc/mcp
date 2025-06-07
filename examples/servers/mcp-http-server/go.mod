module github.com/tmc/mcp/examples/servers/mcp-http-server

go 1.23

require github.com/tmc/mcp v0.0.0-00010101000000-000000000000

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/sourcegraph/jsonrpc2 v0.2.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
)

replace github.com/tmc/mcp => ../../..
