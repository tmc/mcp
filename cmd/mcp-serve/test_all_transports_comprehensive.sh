#!/bin/bash

# Comprehensive test for all MCP transports (stdio, SSE, streamableHttp)

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results directory
RESULTS_DIR="comprehensive_transport_results"
mkdir -p "$RESULTS_DIR"

# Test cases for each transport
TEST_CASES=(
    "initialize|{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"2024-11-05\",\"capabilities\":{},\"clientInfo\":{\"name\":\"ComprehensiveTest\",\"version\":\"1.0.0\"}}}"
    "list_prompts|{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"prompts/list\",\"params\":{}}"
    "list_tools|{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/list\",\"params\":{}}"
    "list_resources|{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"resources/list\",\"params\":{}}"
    "echo_tool|{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"tools/call\",\"params\":{\"name\":\"echo\",\"arguments\":{\"message\":\"Test\"}}}"
)

# Build tools if needed
build_tools() {
    if [ ! -f "./mcp-serve" ]; then
        echo "Building mcp-serve..."
        go build -o mcp-serve main.go
    fi
    
    if [ ! -f "./test_http_client_v2" ]; then
        echo "Building HTTP test client..."
        go build -o test_http_client_v2 test_http_client_v2.go
    fi
}

# Test stdio transport
test_stdio() {
    echo -e "${BLUE}=== Testing STDIO Transport ===${NC}"
    
    local workspace="$RESULTS_DIR/stdio_workspace"
    mkdir -p "$workspace"
    
    # Start server
    ./mcp-serve --workspace="$workspace" -- npx @modelcontextprotocol/server-everything stdio &
    local server_pid=$!
    sleep 3
    
    # Check if server is running
    if ! ./mcp-serve --workspace="$workspace" --status; then
        echo -e "${RED}Failed to start stdio server${NC}"
        return 1
    fi
    
    # Run test cases
    local results_file="$RESULTS_DIR/stdio_results.json"
    echo '{"transport": "stdio", "tests": []}' > "$results_file"
    
    for test_case in "${TEST_CASES[@]}"; do
        IFS='|' read -r name request <<< "$test_case"
        echo -e "${YELLOW}Testing: $name${NC}"
        
        # Send request
        echo "$request" | ./mcp-serve --workspace="$workspace" --send > "$workspace/response.json" 2>/dev/null
        
        # Check response
        if [ -s "$workspace/response.json" ]; then
            echo -e "${GREEN}Response received${NC}"
            # Save result
            response=$(cat "$workspace/response.json")
            jq --arg name "$name" --arg req "$request" --arg resp "$response" \
               '.tests += [{"name": $name, "request": $req, "response": $resp}]' \
               "$results_file" > tmp.$$.json && mv tmp.$$.json "$results_file"
        else
            echo -e "${RED}No response${NC}"
        fi
    done
    
    # Stop server
    ./mcp-serve --workspace="$workspace" --stop
    
    echo ""
}

# Test SSE transport
test_sse() {
    echo -e "${BLUE}=== Testing SSE Transport ===${NC}"
    
    # Start server
    npx @modelcontextprotocol/server-everything sse > "$RESULTS_DIR/sse_server.log" 2>&1 &
    local server_pid=$!
    sleep 5
    
    # Check if server is running
    if ! ps -p $server_pid > /dev/null; then
        echo -e "${RED}Failed to start SSE server${NC}"
        return 1
    fi
    
    # Run test cases
    local results_file="$RESULTS_DIR/sse_results.json"
    echo '{"transport": "sse", "tests": []}' > "$results_file"
    
    for test_case in "${TEST_CASES[@]}"; do
        IFS='|' read -r name request <<< "$test_case"
        echo -e "${YELLOW}Testing: $name${NC}"
        
        # Extract method from request
        method=$(echo "$request" | jq -r '.method')
        params=$(echo "$request" | jq -c '.params')
        
        # Send request via HTTP client
        ./test_http_client_v2 \
            -url="http://localhost:3001" \
            -transport="sse" \
            -method="$method" \
            -params="$params" \
            > "$RESULTS_DIR/sse_${name}.log" 2>&1
        
        # Check if response contains JSON-RPC
        if grep -q "JSON-RPC Response" "$RESULTS_DIR/sse_${name}.log"; then
            echo -e "${GREEN}Valid response received${NC}"
        else
            echo -e "${RED}No valid JSON-RPC response${NC}"
        fi
        
        # Save result
        jq --arg name "$name" --arg req "$request" --arg log "$(cat "$RESULTS_DIR/sse_${name}.log")" \
           '.tests += [{"name": $name, "request": $req, "log": $log}]' \
           "$results_file" > tmp.$$.json && mv tmp.$$.json "$results_file"
    done
    
    # Stop server
    kill $server_pid 2>/dev/null || true
    
    echo ""
}

# Test streamableHttp transport
test_streamable_http() {
    echo -e "${BLUE}=== Testing StreamableHTTP Transport ===${NC}"
    
    # Start server
    npx @modelcontextprotocol/server-everything streamableHttp > "$RESULTS_DIR/http_server.log" 2>&1 &
    local server_pid=$!
    sleep 5
    
    # Check if server is running
    if ! ps -p $server_pid > /dev/null; then
        echo -e "${RED}Failed to start streamableHttp server${NC}"
        return 1
    fi
    
    # Run test cases
    local results_file="$RESULTS_DIR/http_results.json"
    echo '{"transport": "streamableHttp", "tests": []}' > "$results_file"
    
    for test_case in "${TEST_CASES[@]}"; do
        IFS='|' read -r name request <<< "$test_case"
        echo -e "${YELLOW}Testing: $name${NC}"
        
        # Extract method from request
        method=$(echo "$request" | jq -r '.method')
        params=$(echo "$request" | jq -c '.params')
        
        # Send request via HTTP client
        ./test_http_client_v2 \
            -url="http://localhost:3001" \
            -transport="http" \
            -method="$method" \
            -params="$params" \
            > "$RESULTS_DIR/http_${name}.log" 2>&1
        
        # Check if response contains JSON-RPC
        if grep -q "JSON-RPC Response" "$RESULTS_DIR/http_${name}.log"; then
            echo -e "${GREEN}Valid response received${NC}"
        else
            echo -e "${RED}No valid JSON-RPC response${NC}"
        fi
        
        # Save result
        jq --arg name "$name" --arg req "$request" --arg log "$(cat "$RESULTS_DIR/http_${name}.log")" \
           '.tests += [{"name": $name, "request": $req, "log": $log}]' \
           "$results_file" > tmp.$$.json && mv tmp.$$.json "$results_file"
    done
    
    # Stop server
    kill $server_pid 2>/dev/null || true
    
    echo ""
}

# Generate comparison report
generate_report() {
    echo -e "${BLUE}=== Generating Comparison Report ===${NC}"
    
    local report="$RESULTS_DIR/comparison_report.md"
    
    cat > "$report" << EOF
# MCP Transport Comparison Report

## Test Date: $(date)

## Summary

### STDIO Transport
$(if [ -f "$RESULTS_DIR/stdio_results.json" ]; then
    echo "- Total tests: $(jq '.tests | length' "$RESULTS_DIR/stdio_results.json")"
    echo "- Success rate: $(jq '[.tests[].response | select(. != null)] | length' "$RESULTS_DIR/stdio_results.json" 2>/dev/null || echo "0")/$(jq '.tests | length' "$RESULTS_DIR/stdio_results.json")"
else
    echo "- Not tested"
fi)

### SSE Transport
$(if [ -f "$RESULTS_DIR/sse_results.json" ]; then
    echo "- Total tests: $(jq '.tests | length' "$RESULTS_DIR/sse_results.json")"
    echo "- Successful connections: $(grep -l "JSON-RPC Response" "$RESULTS_DIR"/sse_*.log 2>/dev/null | wc -l)"
else
    echo "- Not tested"
fi)

### StreamableHTTP Transport
$(if [ -f "$RESULTS_DIR/http_results.json" ]; then
    echo "- Total tests: $(jq '.tests | length' "$RESULTS_DIR/http_results.json")"
    echo "- Successful connections: $(grep -l "JSON-RPC Response" "$RESULTS_DIR"/http_*.log 2>/dev/null | wc -l)"
else
    echo "- Not tested"
fi)

## Detailed Results

### STDIO Test Results
$(if [ -f "$RESULTS_DIR/stdio_results.json" ]; then
    jq -r '.tests[] | "- \(.name): " + (if .response then "Success" else "Failed" end)' "$RESULTS_DIR/stdio_results.json"
else
    echo "No results available"
fi)

### SSE Test Results
$(if [ -f "$RESULTS_DIR/sse_results.json" ]; then
    for log in "$RESULTS_DIR"/sse_*.log; do
        name=$(basename "$log" .log | sed 's/sse_//')
        status=$(grep -q "JSON-RPC Response" "$log" && echo "Success" || echo "Failed")
        echo "- $name: $status"
    done
else
    echo "No results available"
fi)

### StreamableHTTP Test Results
$(if [ -f "$RESULTS_DIR/http_results.json" ]; then
    for log in "$RESULTS_DIR"/http_*.log; do
        name=$(basename "$log" .log | sed 's/http_//')
        status=$(grep -q "JSON-RPC Response" "$log" && echo "Success" || echo "Failed")
        echo "- $name: $status"
    done
else
    echo "No results available"
fi)

## Conclusions

1. **STDIO**: Best for command-line and process-based integration
2. **SSE**: Suitable for real-time web applications with session management
3. **StreamableHTTP**: Designed for HTTP clients that support streaming responses

## Log Files

All detailed logs are available in: $RESULTS_DIR/
EOF

    echo "Report generated: $report"
}

# Main execution
main() {
    echo -e "${GREEN}Starting Comprehensive Transport Test${NC}"
    echo "Testing all MCP transports with multiple test cases"
    echo ""
    
    # Build required tools
    build_tools
    
    # Run tests
    test_stdio
    test_sse
    test_streamable_http
    
    # Generate report
    generate_report
    
    echo ""
    echo -e "${GREEN}All tests completed!${NC}"
    echo "Results saved to: $RESULTS_DIR/"
    
    # Display summary
    echo ""
    echo "Summary Report:"
    cat "$RESULTS_DIR/comparison_report.md"
}

# Run main
main