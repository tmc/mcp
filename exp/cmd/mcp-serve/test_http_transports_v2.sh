#!/bin/bash

# Test HTTP-based transports with proper HTTP client

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results directory
RESULTS_DIR="http_transport_results"
mkdir -p "$RESULTS_DIR"

# Build HTTP client if needed
if [ ! -f "./test_http_client" ]; then
    echo "Building HTTP test client..."
    go build -o test_http_client test_http_client.go
fi

# Test SSE transport
test_sse_transport() {
    echo -e "${BLUE}=== Testing SSE Transport ===${NC}"
    
    # Start SSE server
    echo "Starting SSE server..."
    npx @modelcontextprotocol/server-everything sse > "$RESULTS_DIR/sse_server.log" 2>&1 &
    SSE_PID=$!
    
    # Wait for server to start
    echo "Waiting for server to start..."
    sleep 5
    
    # Check if server is running
    if ! ps -p $SSE_PID > /dev/null; then
        echo -e "${RED}Failed to start SSE server${NC}"
        cat "$RESULTS_DIR/sse_server.log"
        return 1
    fi
    
    # Test with HTTP client
    echo "Testing SSE endpoint..."
    ./test_http_client -url="http://localhost:3001" -transport="sse" > "$RESULTS_DIR/sse_test.log" 2>&1
    
    # Show results
    echo "SSE Test Results:"
    cat "$RESULTS_DIR/sse_test.log"
    
    # Stop server
    echo "Stopping SSE server..."
    kill $SSE_PID 2>/dev/null || true
    
    # Wait for process to terminate
    sleep 2
    
    echo ""
}

# Test StreamableHttp transport
test_streamable_http() {
    echo -e "${BLUE}=== Testing StreamableHttp Transport ===${NC}"
    
    # Start streamableHttp server
    echo "Starting streamableHttp server..."
    npx @modelcontextprotocol/server-everything streamableHttp > "$RESULTS_DIR/http_server.log" 2>&1 &
    HTTP_PID=$!
    
    # Wait for server to start
    echo "Waiting for server to start..."
    sleep 5
    
    # Check if server is running
    if ! ps -p $HTTP_PID > /dev/null; then
        echo -e "${RED}Failed to start streamableHttp server${NC}"
        cat "$RESULTS_DIR/http_server.log"
        return 1
    fi
    
    # Get port from server log if available
    PORT=$(grep -oE "port [0-9]+" "$RESULTS_DIR/http_server.log" | grep -oE "[0-9]+" | head -1 || echo "3001")
    echo "Server running on port: $PORT"
    
    # Test with HTTP client
    echo "Testing HTTP endpoint..."
    ./test_http_client -url="http://localhost:$PORT" -transport="http" > "$RESULTS_DIR/http_test.log" 2>&1
    
    # Show results
    echo "HTTP Test Results:"
    cat "$RESULTS_DIR/http_test.log"
    
    # Stop server
    echo "Stopping streamableHttp server..."
    kill $HTTP_PID 2>/dev/null || true
    
    # Wait for process to terminate
    sleep 2
    
    echo ""
}

# Test HTTP transport with different methods
test_json_rpc_methods() {
    echo -e "${BLUE}=== Testing JSON-RPC Methods via HTTP ===${NC}"
    
    # Start a server (using streamableHttp)
    echo "Starting test server..."
    npx @modelcontextprotocol/server-everything streamableHttp > "$RESULTS_DIR/methods_server.log" 2>&1 &
    SERVER_PID=$!
    
    sleep 5
    
    # Test different methods
    METHODS=(
        "initialize|{\\\"protocolVersion\\\":\\\"2024-11-05\\\",\\\"capabilities\\\":{},\\\"clientInfo\\\":{\\\"name\\\":\\\"HTTPTest\\\",\\\"version\\\":\\\"1.0.0\\\"}}"
        "prompts/list|{}"
        "tools/list|{}"
        "resources/list|{}"
    )
    
    for method_params in "${METHODS[@]}"; do
        IFS='|' read -r method params <<< "$method_params"
        echo -e "${YELLOW}Testing method: $method${NC}"
        
        ./test_http_client \
            -url="http://localhost:3001" \
            -transport="http" \
            -method="$method" \
            -params="$params" \
            > "$RESULTS_DIR/method_${method//\//_}.log" 2>&1
            
        echo "Response:"
        grep -E "(JSON-RPC Response|Status:|Error:)" "$RESULTS_DIR/method_${method//\//_}.log" || echo "No valid response"
        echo ""
    done
    
    # Stop server
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
}

# Main execution
echo -e "${GREEN}Starting HTTP Transport Tests v2${NC}"
echo "Using custom HTTP client for proper testing"
echo ""

# Run tests
test_sse_transport
test_streamable_http
test_json_rpc_methods

# Generate summary report
SUMMARY="$RESULTS_DIR/summary.md"
cat > "$SUMMARY" << EOF
# HTTP Transport Test Results

## Test Date: $(date)

## SSE Transport
- Server Started: $(grep -q "Server is running" "$RESULTS_DIR/sse_server.log" 2>/dev/null && echo "Yes" || echo "No")
- Connection: $(grep -q "SSE Connection Status: 200" "$RESULTS_DIR/sse_test.log" 2>/dev/null && echo "Success" || echo "Failed")
- Events Received: $(grep -c "SSE:" "$RESULTS_DIR/sse_test.log" 2>/dev/null || echo "0")

## StreamableHttp Transport
- Server Started: $(grep -q "Server is running" "$RESULTS_DIR/http_server.log" 2>/dev/null && echo "Yes" || echo "No")
- Response Status: $(grep "Status:" "$RESULTS_DIR/http_test.log" 2>/dev/null | head -1 || echo "Unknown")

## JSON-RPC Methods Tested
$(for log in "$RESULTS_DIR"/method_*.log; do
    method=$(basename "$log" .log | sed 's/method_//' | sed 's/_/\//g')
    status=$(grep -q "JSON-RPC Response" "$log" && echo "Success" || echo "Failed")
    echo "- $method: $status"
done)

## Server Logs
See individual log files in $RESULTS_DIR/ for details

## Recommendations
1. Check server logs for actual API endpoints
2. Verify JSON-RPC support in HTTP transports
3. Consider using transport-specific client libraries
EOF

echo -e "${GREEN}Test completed!${NC}"
echo "Results saved to: $RESULTS_DIR/"
echo ""
echo "Summary:"
cat "$SUMMARY"