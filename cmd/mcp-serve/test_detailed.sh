#!/bin/bash

# Detailed test script for mcp-serve with the everything server

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
    local expected_pattern="$3"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    echo ""
    echo -e "${BLUE}=== ${description} ===${NC}"
    echo "Request: $request"
    
    # Clear previous response
    > "$TEST_WS/response.json"
    
    # Send request
    echo "$request" | ./mcp-serve --workspace="$TEST_WS" --send > "$TEST_WS/response.json" 2>/dev/null
    
    # Wait for response
    sleep 1
    
    # Check response
    if [ -s "$TEST_WS/response.json" ]; then
        echo -e "${GREEN}Response received:${NC}"
        cat "$TEST_WS/response.json"
        echo ""
        
        # Pattern matching validation
        if [ -n "$expected_pattern" ]; then
            if grep -q "$expected_pattern" "$TEST_WS/response.json"; then
                echo -e "${GREEN}✓ Expected pattern found: $expected_pattern${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}✗ Expected pattern not found: $expected_pattern${NC}"
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        else
            # Just check for valid JSON-RPC
            if grep -q '"jsonrpc":"2.0"' "$TEST_WS/response.json"; then
                echo -e "${GREEN}✓ Valid JSON-RPC response${NC}"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}✗ Invalid JSON-RPC response${NC}"
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
        fi
    else
        echo -e "${RED}✗ No response received${NC}"
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

# === INITIALIZATION TESTS ===
send_request '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"DetailedTestClient","version":"1.0.0"}}}' \
    "Initialize server" \
    "example-servers/everything"

# === PROMPT TESTS ===
send_request '{"jsonrpc":"2.0","id":2,"method":"prompts/list","params":{}}' \
    "List all prompts" \
    "simple_prompt"

send_request '{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"simple_prompt"}}' \
    "Get simple prompt without arguments" \
    "This is a simple prompt"

send_request '{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"complex_prompt","arguments":{"temperature":0.8,"style":"creative"}}}' \
    "Get complex prompt with arguments" \
    "temperature"

send_request '{"jsonrpc":"2.0","id":5,"method":"prompts/get","params":{"name":"resource_prompt","arguments":{"resourceId":42}}}' \
    "Get prompt with resource reference" \
    "resourceId"

# === RESOURCE TESTS ===
send_request '{"jsonrpc":"2.0","id":6,"method":"resources/list","params":{}}' \
    "List resources (first page)" \
    "test://static/resource/1"

send_request '{"jsonrpc":"2.0","id":7,"method":"resources/list","params":{"cursor":"MTA="}}' \
    "List resources (with cursor)" \
    "test://static/resource/"

send_request '{"jsonrpc":"2.0","id":8,"method":"resources/read","params":{"uri":"test://static/resource/1"}}' \
    "Read existing resource" \
    "Resource 1"

send_request '{"jsonrpc":"2.0","id":9,"method":"resources/read","params":{"uri":"test://static/resource/2"}}' \
    "Read resource with blob" \
    "UmVzb3VyY2U"

send_request '{"jsonrpc":"2.0","id":10,"method":"resources/subscribe","params":{"uri":"test://static/resource/1"}}' \
    "Subscribe to resource updates" \
    "subscription"

send_request '{"jsonrpc":"2.0","id":11,"method":"resources/unsubscribe","params":{"uri":"test://static/resource/1"}}' \
    "Unsubscribe from resource" \
    "unsubscribe"

# === TOOL TESTS ===
send_request '{"jsonrpc":"2.0","id":12,"method":"tools/list","params":{}}' \
    "List all tools" \
    "echo"

send_request '{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Hello MCP!"}}}' \
    "Call echo tool" \
    "Hello MCP!"

send_request '{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"add","arguments":{"a":10,"b":25}}}' \
    "Call add tool" \
    "35"

send_request '{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"printEnv"}}' \
    "Call printEnv tool" \
    "ENV"

send_request '{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"getTinyImage"}}' \
    "Call getTinyImage tool" \
    "image"

send_request '{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"annotatedMessage","arguments":{"messageType":"success","includeImage":false}}}' \
    "Call annotatedMessage with success" \
    "success"

send_request '{"jsonrpc":"2.0","id":18,"method":"tools/call","params":{"name":"annotatedMessage","arguments":{"messageType":"error"}}}' \
    "Call annotatedMessage with error" \
    "error"

send_request '{"jsonrpc":"2.0","id":19,"method":"tools/call","params":{"name":"getResourceReference","arguments":{"resourceId":15}}}' \
    "Get resource reference" \
    "test://static/resource/15"

# === LOGGING TESTS ===
send_request '{"jsonrpc":"2.0","id":20,"method":"logging/setLevel","params":{"level":"debug"}}' \
    "Set logging level to debug" \
    "result"

send_request '{"jsonrpc":"2.0","id":21,"method":"logging/setLevel","params":{"level":"error"}}' \
    "Set logging level to error" \
    "result"

# === ERROR TESTS ===
send_request '{"jsonrpc":"2.0","id":22,"method":"invalid/method","params":{}}' \
    "Call invalid method" \
    "Method not found"

send_request '{"jsonrpc":"2.0","id":23,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}' \
    "Call nonexistent tool" \
    "Unknown tool"

send_request '{"jsonrpc":"2.0","id":24,"method":"prompts/get","params":{"name":"nonexistent_prompt"}}' \
    "Get nonexistent prompt" \
    "Unknown prompt"

send_request '{"jsonrpc":"2.0","id":25,"method":"resources/read","params":{"uri":"invalid://uri"}}' \
    "Read invalid resource URI" \
    "Unknown resource"

# === NOTIFICATION TEST ===
echo ""
echo -e "${BLUE}=== Send notification (no response expected) ===${NC}"
echo '{"jsonrpc":"2.0","method":"progress/update","params":{"token":"test-token","progress":75}}' | ./mcp-serve --workspace="$TEST_WS" --send
echo -e "${GREEN}✓ Notification sent${NC}"
TOTAL_TESTS=$((TOTAL_TESTS + 1))
PASSED_TESTS=$((PASSED_TESTS + 1))

# === MALFORMED REQUEST TESTS ===
send_request '{"invalid":"json"}' \
    "Malformed JSON-RPC request" \
    "error"

send_request '{}' \
    "Empty JSON object" \
    "error"

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

# Create test report
REPORT_FILE="$TEST_WS/test_report.md"
cat > "$REPORT_FILE" << EOF
# MCP Server Comprehensive Test Report

## Test Summary
- **Date**: $(date)
- **Server**: npx @modelcontextprotocol/server-everything stdio
- **Total Tests**: $TOTAL_TESTS
- **Passed**: $PASSED_TESTS
- **Failed**: $FAILED_TESTS

## Test Categories

### Initialization
- Server initialization with protocol version 2024-11-05

### Prompts
- List prompts
- Get simple prompt
- Get complex prompt with arguments
- Get prompt with resource reference

### Resources
- List resources (pagination)
- Read text resources
- Read blob resources
- Subscribe/unsubscribe to resources

### Tools
- List available tools
- Call echo tool
- Call add tool (arithmetic)
- Call printEnv tool
- Call getTinyImage tool
- Call annotatedMessage tool (with different message types)
- Call getResourceReference tool

### Logging
- Set logging levels

### Error Handling
- Invalid method calls
- Nonexistent tools/prompts/resources
- Malformed requests

### Notifications
- Send progress notifications

## Server Capabilities
$(cat "$TEST_WS/.mcp-server.stdout" | grep -A 10 "capabilities" | head -15)

EOF

echo ""
echo "Test report created: $REPORT_FILE"

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

echo "Test workspace kept for inspection: $TEST_WS"
echo -e "${GREEN}Detailed test completed!${NC}"