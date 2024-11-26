package mcptest

import (
	"context"
	"os"
	"testing"
)

func TestRunTXTARFile(t *testing.T) {
	const testScript = `Test basic MCP functionality
-- script.txt --
# Test initialization
mcp ./testdata/echo-server initialize {"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}
stdout '{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"echo","version":"1.0.0"}}'

# Test tool listing
mcp ./testdata/echo-server tools/list {}
stdout '{"tools":[{"name":"echo","description":"Echo the input","inputSchema":{"type":"object","properties":{"message":{"type":"string"}}}}]}'

# Test tool call
mcp ./testdata/echo-server tools/call {"name":"echo","arguments":{"message":"hello"}}
stdout '{"content":[{"type":"text","text":"hello"}]}'

# Test invalid tool
! mcp ./testdata/echo-server tools/call {"name":"invalid","arguments":{}}
stderr 'unknown tool: invalid'

# Test with conditions
[unix] mcp ./testdata/echo-server tools/list {}
[windows] skip 'not supported on windows'

# Test with optional command
? mcp ./testdata/echo-server tools/call {"name":"optional","arguments":{}}
`

	if err := os.WriteFile("test.txtar", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("test.txtar")

	if err := RunTxTarFile(context.Background(), "test.txtar", "."); err != nil {
		t.Fatal(err)
	}
}
