module github.com/tmc/mcp/exp/adapters

go 1.23

require (
	github.com/mark3labs/mcp-go v0.28.0
	github.com/tmc/mcp v0.0.0-00010101000000-000000000000
)

// Replace with local version for development
replace github.com/tmc/mcp => ../../

// Dependencies will be inherited from the main mcp module