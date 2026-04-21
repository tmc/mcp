module github.com/tmc/mcp/examples/servers/mcp-time-server

go 1.24.0

toolchain go1.24.3

require github.com/tmc/mcp v0.0.0-20000101000000-000000000000

require (
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	rsc.io/script v0.0.2 // indirect
)

replace github.com/tmc/mcp => ../../..
