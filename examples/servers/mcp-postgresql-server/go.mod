module github.com/tmc/mcp/examples/servers/mcp-postgresql-server

go 1.22

require (
	github.com/lib/pq v1.10.9
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
)

replace github.com/tmc/mcp => ../../..
