module github.com/tmc/mcp/exp

go 1.25.0

require (
	github.com/tmc/mcp v0.0.0
	github.com/tmc/mcp/testing/mcptestutil v0.0.0
)

require (
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/exp/event v0.0.0-20260410095643-746e56fc9e2f // indirect
	golang.org/x/exp/jsonrpc2 v0.0.0-20260529124908-c761662dc8c9 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/term v0.33.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	rsc.io/script v0.0.2 // indirect
)

replace github.com/tmc/mcp => ..

replace github.com/tmc/mcp/testing/mcptestutil => ../testing/mcptestutil
