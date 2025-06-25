module github.com/tmc/mcp

go 1.23.0

toolchain go1.24.2

require github.com/google/go-cmp v0.6.0

require (
	github.com/gorilla/websocket v1.4.1
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/term v0.32.0
)

require (
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	rsc.io/uncover v0.0.2 // indirect
)

tool (
	github.com/tmc/mcp/cmd/mcpdiff
	rsc.io/uncover
)
