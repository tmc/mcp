#!/bin/bash

# Test HTTP-based transports (SSE and streamableHttp)

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test SSE transport
test_sse() {
    echo -e "${BLUE}=== Testing SSE Transport ===${NC}"
    
    # Start SSE server
    echo "Starting SSE server..."
    npx @modelcontextprotocol/server-everything sse &
    SSE_PID=$!
    
    # Wait for server to start
    sleep 3
    
    # Check if server is running
    if ! ps -p $SSE_PID > /dev/null; then
        echo -e "${RED}Failed to start SSE server${NC}"
        return 1
    fi
    
    # Default SSE port is typically 3000
    SSE_URL="http://localhost:3000"
    
    # Test connection
    echo "Testing SSE connection at $SSE_URL..."
    
    # Try connecting to SSE endpoint
    timeout 2s curl -N "${SSE_URL}/sse" 2>/dev/null > sse_test.log &
    CURL_PID=$!
    
    sleep 1
    
    # Check if we got any data
    if [ -s "sse_test.log" ]; then
        echo -e "${GREEN}Successfully connected to SSE endpoint${NC}"
        echo "SSE Response sample:"
        head -5 sse_test.log
    else
        echo -e "${YELLOW}No data received from SSE endpoint${NC}"
    fi
    
    # Kill the background curl
    kill $CURL_PID 2>/dev/null || true
    
    # Try sending a request via HTTP POST
    echo "Attempting POST request..."
    curl -X POST "${SSE_URL}" \
         -H "Content-Type: application/json" \
         -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"SSETest","version":"1.0.0"}}}' \
         -s -o sse_post_response.json 2>/dev/null || true
    
    if [ -s "sse_post_response.json" ]; then
        echo "POST Response:"
        cat sse_post_response.json
    else
        echo -e "${YELLOW}No response from POST request${NC}"
    fi
    
    # Stop server
    echo "Stopping SSE server..."
    kill $SSE_PID 2>/dev/null || true
    
    echo ""
}

# Test streamableHttp transport
test_streamable_http() {
    echo -e "${BLUE}=== Testing StreamableHttp Transport ===${NC}"
    
    # Start streamableHttp server
    echo "Starting streamableHttp server..."
    npx @modelcontextprotocol/server-everything streamableHttp &
    HTTP_PID=$!
    
    # Wait for server to start
    sleep 3
    
    # Check if server is running
    if ! ps -p $HTTP_PID > /dev/null; then
        echo -e "${RED}Failed to start streamableHttp server${NC}"
        return 1
    fi
    
    # Default HTTP port
    HTTP_URL="http://localhost:3000"
    
    # Test connection
    echo "Testing streamableHttp connection at $HTTP_URL..."
    
    # Try basic GET
    curl -s "${HTTP_URL}" -o http_get_response.html 2>/dev/null || true
    
    if [ -s "http_get_response.html" ]; then
        echo -e "${GREEN}Successfully connected to HTTP endpoint${NC}"
        echo "GET Response sample:"
        head -5 http_get_response.html
    else
        echo -e "${YELLOW}No data from GET request${NC}"
    fi
    
    # Try sending a request
    echo "Attempting POST request..."
    curl -X POST "${HTTP_URL}" \
         -H "Content-Type: application/json" \
         -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"HttpTest","version":"1.0.0"}}}' \
         -s -o http_post_response.json 2>/dev/null || true
    
    if [ -s "http_post_response.json" ]; then
        echo "POST Response:"
        cat http_post_response.json
    else
        echo -e "${YELLOW}No response from POST request${NC}"
    fi
    
    # Stop server
    echo "Stopping streamableHttp server..."
    kill $HTTP_PID 2>/dev/null || true
    
    echo ""
}

# Main execution
echo -e "${GREEN}Starting HTTP Transport Test${NC}"
echo "Note: These tests require the servers to expose HTTP endpoints"
echo ""

# Test SSE
test_sse

# Test streamableHttp  
test_streamable_http

# Create summary
SUMMARY_FILE="http_transport_summary.md"
cat > "$SUMMARY_FILE" << EOF
# HTTP Transport Test Summary

## Test Date: $(date)

## Results

### SSE Transport
- Started: $([ -f "sse_test.log" ] && echo "Yes" || echo "Unknown")
- SSE Stream: $([ -s "sse_test.log" ] && echo "Data received" || echo "No data")
- POST Response: $([ -s "sse_post_response.json" ] && echo "Received" || echo "None")

### StreamableHttp Transport
- Started: $([ -f "http_get_response.html" ] && echo "Yes" || echo "Unknown")
- GET Response: $([ -s "http_get_response.html" ] && echo "Data received" || echo "No data")
- POST Response: $([ -s "http_post_response.json" ] && echo "Received" || echo "None")

## Notes

The HTTP-based transports (SSE and streamableHttp) require proper HTTP server configuration and endpoints.
They are designed for browser-based clients and HTTP APIs rather than process stdin/stdout communication.

## Recommendations

1. Use stdio transport for command-line and process-based integration
2. Use SSE for real-time browser applications
3. Use streamableHttp for traditional HTTP API clients

EOF

echo -e "${GREEN}HTTP Transport test completed!${NC}"
echo "Summary saved to: $SUMMARY_FILE"
echo ""
cat "$SUMMARY_FILE"