package mcptest

import (
	"bytes"
	"context"
	"os"
	"testing"
)

func TestRunTXTARFile(t *testing.T) {
	const testScript = `Test basic MCP functionality
-- script.txt --
# Test initialization
mcp ./testdata/echo-server initialize {}
stdout '{"jsonrpc":"2.0","id":1,"result":{"name":"echo","version":"1.0.0","protocolVersion":"2024-11-05"}}'

# Test tool listing
mcp ./testdata/echo-server listTools {}
stdout '{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo the input"}]}}'

# Test tool call
mcp ./testdata/echo-server echo {"message":"hello"}
stdout '{"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"hello"}]}}'

# Test invalid tool
! mcp ./testdata/echo-server invalid {}
stderr 'unknown method: invalid'

# Test with conditions
[unix] mcp ./testdata/echo-server listTools {}
[windows] skip 'not supported on windows'

# Test with optional command
? mcp ./testdata/echo-server optional {}
`

	if err := os.WriteFile("test.txtar", []byte(testScript), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("test.txtar")

	var output bytes.Buffer
	if err := RunTxTarFile(context.Background(), "test.txtar", &output); err != nil {
		t.Fatal(err)
	}
}
