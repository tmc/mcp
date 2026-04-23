#!/bin/bash

# Test script for mcp-proxy

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Testing mcp-proxy${NC}"

# Build
echo "Building mcp-proxy..."
go build -o mcp-proxy main.go

# Test 1: Basic stdio proxy
echo ""
echo -e "${BLUE}Test 1: Basic STDIO proxy${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
    timeout 2s ./mcp-proxy -- npx @modelcontextprotocol/server-everything stdio | \
    grep -q '"result"' && echo -e "${GREEN}✓ Passed${NC}" || echo -e "${RED}✗ Failed${NC}"

# Test 2: Verbose mode
echo ""
echo -e "${BLUE}Test 2: Verbose mode${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | \
    timeout 2s ./mcp-proxy -v -- npx @modelcontextprotocol/server-everything stdio | \
    grep -q '"tools"' && echo -e "${GREEN}✓ Passed${NC}" || echo -e "${RED}✗ Failed${NC}"

# Test 3: Timestamps
echo ""
echo -e "${BLUE}Test 3: Timestamps${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
    timeout 2s ./mcp-proxy -t -- npx @modelcontextprotocol/server-everything stdio | \
    grep -q '\[.*\]' && echo -e "${GREEN}✓ Passed${NC}" || echo -e "${RED}✗ Failed${NC}"

# Test 4: Combined with mcp-connect
echo ""
echo -e "${BLUE}Test 4: Integration with mcp-connect${NC}"
if [ -f "../mcp-connect/mcp-connect" ]; then
    ../mcp-connect/mcp-connect -cmd="./mcp-proxy -v -- npx @modelcontextprotocol/server-everything stdio" \
        -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' | \
        grep -q '"result"' && echo -e "${GREEN}✓ Passed${NC}" || echo -e "${RED}✗ Failed${NC}"
else
    echo -e "${RED}✗ Skipped (mcp-connect not found)${NC}"
fi

# Demo usage
echo ""
echo -e "${BLUE}Demo Usage:${NC}"
echo ""
echo "1. Basic proxy:"
echo "   ./mcp-proxy -- npx @modelcontextprotocol/server-everything stdio"
echo ""
echo "2. Verbose with timestamps:"
echo "   ./mcp-proxy -v -t -- node my-server.js"
echo ""
echo "3. HTTP proxy:"
echo "   ./mcp-proxy -transport=http -listen=:8080 -target=http://localhost:3001"
echo ""
echo "4. With mcp-connect:"
echo '   mcp-connect -cmd="./mcp-proxy -v -- npx @modelcontextprotocol/server-everything stdio"'