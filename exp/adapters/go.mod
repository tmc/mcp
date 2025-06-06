module github.com/tmc/mcp/exp/adapters

go 1.23.0

toolchain go1.24.3

require (
	github.com/mark3labs/mcp-go v0.28.0
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

// Replace with local version for development
replace github.com/tmc/mcp => ../../

// Dependencies will be inherited from the main mcp module
