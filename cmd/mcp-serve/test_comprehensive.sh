#!/bin/bash

# Comprehensive test script for mcp-serve with the everything server

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test state
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper function to send request and get response
send_request() {
    local request="$1"
    local description="$2"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo ""
    echo -e "${BLUE}=== ${description} ===${NC}"
    echo "Request: $request"
    
    # Send request
    echo "$request" | ./mcp-serve --workspace="$TEST_WS" -v --send > "$TEST_WS/response.json" 2>"$TEST_WS/send_debug.log"
    
    # Wait for response
    sleep 1
    
    # Check response
    if [ -s "$TEST_WS/response.json" ]; then
        echo -e "${GREEN}Response received:${NC}"
        cat "$TEST_WS/response.json"
        echo ""
        
        # Basic validation
        if grep -q '"jsonrpc":"2.0"' "$TEST_WS/response.json"; then
            echo -e "${GREEN}✓ Valid JSON-RPC response${NC}"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        else
            echo -e "${RED}✗ Invalid JSON-RPC response${NC}"
            FAILED_TESTS=$((FAILED_TESTS + 1))
        fi
    else
        echo -e "${RED}✗ No response received${NC}"
        cat "$TEST_WS/send_debug.log"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
}

echo "Building mcp-serve..."
cd /Volumes/tmc/go/src/github.com/tmc/mcp/cmd/mcp-serve
go build -o mcp-serve

echo -e "${GREEN}Built mcp-serve successfully${NC}"

# Create a test workspace
TEST_WS="/tmp/mcp-serve-test-$$"
mkdir -p "$TEST_WS"

echo "Starting MCP server in workspace: $TEST_WS"
./mcp-serve --workspace="$TEST_WS" -v -- npx @modelcontextprotocol/server-everything stdio &
MCP_PID=$!

# Give the server time to start fully
sleep 3

echo "Checking server status..."
if ./mcp-serve --workspace="$TEST_WS" --status; then
    echo -e "${GREEN}Server is running${NC}"
else
    echo -e "${RED}Server is not running${NC}"
    exit 1
fi

# 1. Initialize
send_request '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"ComprehensiveTestClient","version":"1.0.0"}}}' \
    "Test 1: Initialize"

# 2. List prompts
send_request '{"jsonrpc":"2.0","id":2,"method":"prompts/list","params":{}}' \
    "Test 2: List prompts"

# 3. Get a specific prompt (assuming there's one called "simple_prompt")
send_request '{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"simple_prompt","arguments":{"message":"Hello"}}}' \
    "Test 3: Get prompt"

# 4. List resources
send_request '{"jsonrpc":"2.0","id":4,"method":"resources/list","params":{}}' \
    "Test 4: List resources"

# 5. Read a resource (assuming there's one)
send_request '{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"file:///example.txt"}}' \
    "Test 5: Read resource"

# 6. Subscribe to resource updates
send_request '{"jsonrpc":"2.0","id":6,"method":"resources/subscribe","params":{"uri":"file:///example.txt"}}' \
    "Test 6: Subscribe to resource"

# 7. List tools
send_request '{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{}}' \
    "Test 7: List tools"

# 8. Call a tool (assuming there's a calculator tool)
send_request '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"calculate","arguments":{"operation":"add","a":5,"b":3}}}' \
    "Test 8: Call tool"

# 9. Set logging level
send_request '{"jsonrpc":"2.0","id":9,"method":"logging/setLevel","params":{"level":"info"}}' \
    "Test 9: Set logging level"

# 10. List logs
send_request '{"jsonrpc":"2.0","id":10,"method":"logging/list","params":{}}' \
    "Test 10: List logs"

# 11. Get completion (if supported)
send_request '{"jsonrpc":"2.0","id":11,"method":"completions/complete","params":{"ref":{"type":"ref/resource","uri":"file:///example.txt"},"argument":{"name":"prefix","value":"Hello"}}}' \
    "Test 11: Get completion"

# 12. Send notifications (no response expected)
echo ""
echo -e "${BLUE}=== Test 12: Send notification ===${NC}"
echo '{"jsonrpc":"2.0","method":"notifications/progress","params":{"token":"test-token","progress":50,"total":100}}' | ./mcp-serve --workspace="$TEST_WS" --send
echo -e "${GREEN}✓ Notification sent${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
PASSED_TESTS=$((PASSED_TESTS + 1))

# 13. Test error handling
send_request '{"jsonrpc":"2.0","id":13,"method":"invalid/method","params":{}}' \
    "Test 13: Invalid method (error handling)"

# 14. Test batch request
send_request '[{"jsonrpc":"2.0","id":14,"method":"prompts/list","params":{}},{"jsonrpc":"2.0","id":15,"method":"tools/list","params":{}}]' \
    "Test 14: Batch request"

# Summary
echo ""
echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}================================${NC}"
echo "Total tests: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
else
    echo -e "${YELLOW}Some tests failed${NC}"
fi

echo ""
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

echo -e "${GREEN}Comprehensive test completed!${NC}"