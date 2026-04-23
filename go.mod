module github.com/tmc/mcp

go 1.25.0

toolchain go1.25.9

require github.com/google/go-cmp v0.6.0

require (
	github.com/gorilla/websocket v1.4.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	golang.org/x/crypto v0.39.0
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/time v0.12.0
	golang.org/x/tools v0.33.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	rsc.io/script v0.0.2
)

require (
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/sys v0.36.0 // indirect
	rsc.io/uncover v0.0.2 // indirect
)

tool rsc.io/uncover
