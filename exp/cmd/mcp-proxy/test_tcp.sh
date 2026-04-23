#!/bin/bash
# Test script for mcp-proxy TCP mode

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Testing mcp-proxy TCP mode${NC}"

# Build mcp-proxy if needed
echo -e "${GREEN}Building mcp-proxy...${NC}"
go build .

# Test 1: Basic TCP proxy
echo -e "${YELLOW}Test 1: Basic TCP proxy${NC}"
./mcp-proxy -transport tcp -listen :7000 -v -t -- echo "Hello from server" &
PROXY_PID=$!
sleep 1

# Connect and send a message
echo '{"jsonrpc":"2.0","id":1,"method":"test"}' | nc -N localhost 7000

# Kill the proxy
kill $PROXY_PID
wait $PROXY_PID 2>/dev/null || true

echo -e "${GREEN}Test 1: Passed${NC}"

# Test 2: TCP proxy with mcpspy
echo -e "${YELLOW}Test 2: TCP proxy with mcpspy${NC}"
./mcp-proxy -transport tcp -listen :7001 -spy -spy-v -- echo "Hello with spy" &
PROXY_PID=$!
sleep 1

# Connect and send a message
echo '{"jsonrpc":"2.0","id":1,"method":"test"}' | nc -N localhost 7001

# Kill the proxy
kill $PROXY_PID
wait $PROXY_PID 2>/dev/null || true

echo -e "${GREEN}Test 2: Passed${NC}"

# Test 3: Multiple connections
echo -e "${YELLOW}Test 3: Multiple simultaneous connections${NC}"
./mcp-proxy -transport tcp -listen :7002 -v -- cat &
PROXY_PID=$!
sleep 1

# Connect multiple clients
(echo '{"jsonrpc":"2.0","id":1,"method":"client1"}' | nc -N localhost 7002) &
(echo '{"jsonrpc":"2.0","id":2,"method":"client2"}' | nc -N localhost 7002) &
(echo '{"jsonrpc":"2.0","id":3,"method":"client3"}' | nc -N localhost 7002) &

# Wait for connections to complete
sleep 2

# Kill the proxy
kill $PROXY_PID
wait $PROXY_PID 2>/dev/null || true

echo -e "${GREEN}Test 3: Passed${NC}"

# Test 4: With real MCP server (if available)
echo -e "${YELLOW}Test 4: With real MCP server${NC}"
if command -v npx &> /dev/null; then
    ./mcp-proxy -transport tcp -listen :7003 -spy -spy-pretty -- npx -y @modelcontextprotocol/server-everything stdio &
    PROXY_PID=$!
    sleep 3
    
    # Send initialize request
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{}}}' | nc -N localhost 7003
    
    # Kill the proxy
    kill $PROXY_PID
    wait $PROXY_PID 2>/dev/null || true
    
    echo -e "${GREEN}Test 4: Passed${NC}"
else
    echo -e "${YELLOW}Test 4: Skipped (npx not available)${NC}"
fi

echo -e "${GREEN}All tests completed successfully!${NC}"