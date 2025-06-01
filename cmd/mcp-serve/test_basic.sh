#!/bin/bash

# Basic test script for mcp-serve with the everything server

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "Building mcp-serve..."
cd /Volumes/tmc/go/src/github.com/tmc/mcp/cmd/mcp-serve
go build -o mcp-serve

echo -e "${GREEN}Built mcp-serve successfully${NC}"

# Create a test workspace
TEST_WS="/tmp/mcp-serve-test-$$"
mkdir -p "$TEST_WS"

echo "Starting MCP server in workspace: $TEST_WS"
./mcp-serve --workspace="$TEST_WS" -v -- npx @modelcontextprotocol/server-everything stdio &

# Give the server time to start
sleep 2

echo "Checking server status..."
if ./mcp-serve --workspace="$TEST_WS" --status; then
    echo -e "${GREEN}Server is running${NC}"
else
    echo -e "${RED}Server is not running${NC}"
    exit 1
fi

echo "Sending initialize request..."
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"TestClient","version":"1.0.0"}}}' | ./mcp-serve --workspace="$TEST_WS" -v --send > "$TEST_WS/response.json"

echo "Response from server:"
cat "$TEST_WS/response.json"

echo ""
echo "Checking response validity..."
if grep -q '"jsonrpc":"2.0"' "$TEST_WS/response.json" && grep -q '"id":1' "$TEST_WS/response.json"; then
    echo -e "${GREEN}Response is valid JSON-RPC${NC}"
else
    echo -e "${RED}Invalid response received${NC}"
fi

echo "Stopping server..."
./mcp-serve --workspace="$TEST_WS" --stop

echo ""
echo "Checking if server has stopped..."
if ! ./mcp-serve --workspace="$TEST_WS" --status 2>/dev/null; then
    echo -e "${GREEN}Server stopped successfully${NC}"
else
    echo -e "${RED}Server still running${NC}"
    exit 1
fi

echo "Cleaning up..."
rm -rf "$TEST_WS"

echo -e "${GREEN}Test completed successfully!${NC}"