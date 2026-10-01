// Package mcptrace reads, writes, and records MCP JSON-RPC traces.
//
// Each record has a direction, one JSON value, and an optional timestamp:
//
//	mcp-send {"jsonrpc":"2.0","id":1,"method":"ping"} # 1234567890.123
//
// Blank lines and lines beginning with # are ignored. Optional fields after
// the timestamp are preserved as metadata. Stream recorders also recognize
// server-sent events whose data lines contain JSON values.
package mcptrace
