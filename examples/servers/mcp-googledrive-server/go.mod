module github.com/tmc/mcp/examples/servers/mcp-googledrive-server

go 1.22

require (
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
	golang.org/x/oauth2 v0.15.0
	google.golang.org/api v0.150.0
)

replace github.com/tmc/mcp => ../../..
