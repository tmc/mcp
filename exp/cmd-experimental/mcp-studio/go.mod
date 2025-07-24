module github.com/tmc/mcp/cmd/mcp-studio

go 1.21

require (
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.5.0
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
)

replace github.com/tmc/mcp => ../..