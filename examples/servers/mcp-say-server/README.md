# MCP Say Server

A simple MCP server that invokes macOS's `say` command to perform text-to-speech.

## Usage

```bash
go build .
./mcp-say-server
```

## Example

```bash
echo '{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"say","arguments":{"text":"Hello world"}}}' | ./mcp-say-server
```

## Parameters

- `text` (required): The text to speak
- `voice` (optional): The voice to use (e.g., "Alex", "Samantha")
- `rate` (optional): Speech rate in words per minute