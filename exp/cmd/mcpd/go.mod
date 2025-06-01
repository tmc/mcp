module github.com/tmc/mcp/exp/cmd/mcpd

go 1.23.0

toolchain go1.24.3

require (
	golang.org/x/crypto v0.33.0
	golang.org/x/oauth2 v0.23.0
	golang.org/x/term v0.32.0
)

require (
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/tmc/mcp => ../../..
