#!/bin/bash
# Test client for mcp-proxy TCP mode

HOST=${1:-localhost}
PORT=${2:-7000}

echo "Connecting to $HOST:$PORT"
echo "Sending MCP initialize request..."

# Send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{}}}' | nc -N $HOST $PORT

echo ""
echo "Response received. Connection closed."