module github.com/tmc/mcp/examples/servers/mcp-screencapture-server

go 1.25.0

require (
	github.com/tmc/macgo v0.0.0-00010101000000-000000000000
	github.com/tmc/mcp v0.0.0-20241126155658-6dc8f6842a0a
)

require (
	github.com/ebitengine/purego v0.11.0-alpha.3 // indirect
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp/event v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/exp/jsonrpc2 v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
)

replace github.com/tmc/mcp => ../../..

replace github.com/tmc/macgo => ../../../../macgo
