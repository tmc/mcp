module github.com/tmc/mcp/examples/servers/mcp-github-server

go 1.22

require (
	github.com/google/go-github/v57 v57.0.0
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.15.0
)

replace github.com/tmc/mcp => ../../..
