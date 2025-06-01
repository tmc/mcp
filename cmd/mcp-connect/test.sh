#!/bin/bash

# Test script for mcp-connect

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Testing mcp-connect with all transports${NC}"

# Build if needed
if [ ! -f "./mcp-connect" ]; then
    echo "Building mcp-connect..."
    go build -o mcp-connect main.go
fi

# Test STDIO
echo ""
echo -e "${BLUE}1. Testing STDIO transport${NC}"
RESULT=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./mcp-connect -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>/dev/null)
if echo "$RESULT" | grep -q '"result"'; then
    echo -e "${GREEN}✓ STDIO test passed${NC}"
else
    echo -e "${RED}✗ STDIO test failed${NC}"
    echo "$RESULT"
fi

# Test SSE
echo ""
echo -e "${BLUE}2. Testing SSE transport${NC}"
# Start SSE server
npx @modelcontextprotocol/server-everything sse > sse.log 2>&1 &
SSE_PID=$!
sleep 5

RESULT=$(./mcp-connect -transport=sse -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>/dev/null || echo "failed")
if echo "$RESULT" | grep -q '"result"'; then
    echo -e "${GREEN}✓ SSE test passed${NC}"
else
    echo -e "${RED}✗ SSE test failed${NC}"
    echo "$RESULT"
fi

kill $SSE_PID 2>/dev/null || true

# Test HTTP
echo ""
echo -e "${BLUE}3. Testing HTTP transport${NC}"
# Start HTTP server
npx @modelcontextprotocol/server-everything streamableHttp > http.log 2>&1 &
HTTP_PID=$!
sleep 5

RESULT=$(./mcp-connect -transport=http -request='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}' 2>/dev/null || echo "failed")
if echo "$RESULT" | grep -q '"result"'; then
    echo -e "${GREEN}✓ HTTP test passed${NC}"
else
    echo -e "${RED}✗ HTTP test failed${NC}"
    echo "$RESULT"
fi

kill $HTTP_PID 2>/dev/null || true

# Test script mode
echo ""
echo -e "${BLUE}4. Testing script mode${NC}"
cat > test-requests.txt << EOF
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
EOF

RESULT=$(./mcp-connect -script=test-requests.txt 2>/dev/null)
if echo "$RESULT" | grep -q '"tools"'; then
    echo -e "${GREEN}✓ Script mode test passed${NC}"
else
    echo -e "${RED}✗ Script mode test failed${NC}"
fi

rm -f test-requests.txt

echo ""
echo -e "${GREEN}Test completed!${NC}"