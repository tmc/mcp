#!/bin/bash

# Detailed SSE transport test

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== SSE Transport Detailed Test ===${NC}"

# Start SSE server
echo "Starting SSE server..."
npx @modelcontextprotocol/server-everything sse > sse_server.log 2>&1 &
SERVER_PID=$!

# Wait for server
sleep 5

# Check if running
if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}Server failed to start${NC}"
    cat sse_server.log
    exit 1
fi

echo -e "${GREEN}Server started successfully${NC}"

# Test different methods
METHODS=(
    "initialize"
    "prompts/list"
    "tools/list"
    "resources/list"
)

for method in "${METHODS[@]}"; do
    echo ""
    echo -e "${YELLOW}Testing method: $method${NC}"
    
    # Prepare params based on method
    if [ "$method" = "initialize" ]; then
        params='{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"SSETest","version":"1.0.0"}}'
    else
        params='{}'
    fi
    
    # Run test
    ./test_http_client_v3 \
        -transport=sse \
        -method="$method" \
        -params="$params" \
        -timeout=5s
    
    echo ""
    echo "---"
done

# Stop server
echo -e "${BLUE}Stopping server...${NC}"
kill $SERVER_PID 2>/dev/null || true

echo -e "${GREEN}Test completed${NC}"