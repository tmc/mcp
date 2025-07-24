module github.com/tmc/mcp

go 1.23.0

toolchain go1.24.2

require github.com/google/go-cmp v0.6.0

require (
	github.com/gorilla/websocket v1.4.1
	golang.org/x/exp/jsonrpc2 v0.0.0-20250506013437-ce4c2cf36ca6
	golang.org/x/term v0.33.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v5 v5.3.1 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/exp/event v0.0.0-20220217172124-1812c5b45e43 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/time v0.12.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	rsc.io/uncover v0.0.2 // indirect
)

tool (
	github.com/tmc/mcp/cmd/mcpdiff
	rsc.io/uncover
)
