module github.com/tmc/mcp/examples/servers/mcp-screencapture-server

go 1.24

require (
	github.com/tmc/macgo v0.0.0-00010101000000-000000000000
	github.com/tmc/mcp v0.0.0-20241126155658-6dc8f6842a0a
)

require (
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/tmc/mcp/testing/mcpscripttest v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)

replace github.com/tmc/mcp/testing/mcpscripttest => ../../../testing/mcpscripttest

replace github.com/tmc/mcp => ../../..

replace github.com/tmc/macgo => ../../../../macgo
