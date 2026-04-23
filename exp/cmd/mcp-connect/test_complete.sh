#!/bin/bash

# Complete test for mcp-connect

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Complete test of mcp-connect${NC}"

# Build
echo "Building mcp-connect..."
go build -o mcp-connect main.go

# Kill any existing servers
echo "Cleaning up existing servers..."
pkill -f "@modelcontextprotocol/server-everything" || true
sleep 2

# Test 1: STDIO with single request
echo ""
echo -e "${BLUE}Test 1: STDIO with single request${NC}"
RESULT=$(./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>/dev/null)
if echo "$RESULT" | grep -q '"result"'; then
    echo -e "${GREEN}✓ Passed${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
fi

# Test 2: SSE transport
echo ""
echo -e "${BLUE}Test 2: SSE transport${NC}"
npx @modelcontextprotocol/server-everything sse > /dev/null 2>&1 &
SSE_PID=$!
sleep 5

if ps -p $SSE_PID > /dev/null; then
    RESULT=$(./mcp-connect -transport=sse -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>&1)
    if echo "$RESULT" | grep -q '"result"'; then
        echo -e "${GREEN}✓ Passed${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
        echo "$RESULT"
    fi
else
    echo -e "${RED}✗ Failed to start SSE server${NC}"
fi

kill $SSE_PID 2>/dev/null || true
sleep 2

# Test 3: HTTP transport
echo ""
echo -e "${BLUE}Test 3: HTTP transport${NC}"
npx @modelcontextprotocol/server-everything streamableHttp > /dev/null 2>&1 &
HTTP_PID=$!
sleep 5

if ps -p $HTTP_PID > /dev/null; then
    RESULT=$(./mcp-connect -transport=http -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>&1)
    if echo "$RESULT" | grep -q '"result"'; then
        echo -e "${GREEN}✓ Passed${NC}"
    else
        echo -e "${RED}✗ Failed${NC}"
        echo "$RESULT"
    fi
else
    echo -e "${RED}✗ Failed to start HTTP server${NC}"
fi

kill $HTTP_PID 2>/dev/null || true
sleep 2

# Test 4: Script mode with multiple requests
echo ""
echo -e "${BLUE}Test 4: Script mode${NC}"
cat > test_script.txt << EOF
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Hello from script"}}}
EOF

RESULT=$(./mcp-connect -script=test_script.txt 2>/dev/null)
if echo "$RESULT" | grep -q "Hello from script"; then
    echo -e "${GREEN}✓ Passed${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
fi

rm -f test_script.txt

# Test 5: Different server command
echo ""
echo -e "${BLUE}Test 5: Custom stdio command${NC}"
RESULT=$(./mcp-connect -cmd="npx @modelcontextprotocol/server-everything stdio" -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' 2>/dev/null)
if echo "$RESULT" | grep -q '"result"'; then
    echo -e "${GREEN}✓ Passed${NC}"
else
    echo -e "${RED}✗ Failed${NC}"
fi

echo ""
echo -e "${GREEN}All tests completed!${NC}"

# Demo usage
echo ""
echo -e "${BLUE}Demo Usage:${NC}"
echo ""
echo "1. Interactive mode (stdio):"
echo "   ./mcp-connect"
echo ""
echo "2. Single request (any transport):"
echo "   ./mcp-connect -transport=http -request='{...}'"
echo ""
echo "3. Script mode:"
echo "   ./mcp-connect -script=requests.txt"
echo ""
echo "4. Custom command:"
echo "   ./mcp-connect -cmd='node my-server.js' -request='{...}'"