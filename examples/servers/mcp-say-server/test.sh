#!/bin/bash

echo "Testing MCP Say Server..."

# Simple test command
echo '{"jsonrpc":"2.0","method":"tools/call","id":1,"params":{"name":"say","arguments":{"text":"Hello from MCP Say Server!"}}}' | ./mcp-say-server