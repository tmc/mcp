module github.com/tmc/mcp

go 1.25.0

toolchain go1.25.9

require github.com/google/go-cmp v0.6.0

require (
	github.com/gorilla/websocket v1.4.1
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
	golang.org/x/crypto v0.39.0
	golang.org/x/exp/jsonrpc2 v0.0.0-20260529124908-c761662dc8c9
	golang.org/x/time v0.12.0
	golang.org/x/tools v0.33.0
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da
	rsc.io/script v0.0.2
)

require (
	golang.org/x/exp/event v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/sys v0.43.0 // indirect
	rsc.io/uncover v0.0.2 // indirect
)

tool rsc.io/uncover
