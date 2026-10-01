// Package mcptest connects MCP SDK clients and servers for in-memory tests.
//
// Connect registers cleanup with the test. Supply an SDK client configured with
// callbacks to observe progress, sampling, or elicitation. The fixture exposes
// SDK sessions directly; handlers can use channels to synchronize tests without
// sleeps. Handlers must honor cancellation so that graceful cleanup can finish.
package mcptest
