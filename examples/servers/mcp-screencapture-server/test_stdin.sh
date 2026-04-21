#!/bin/bash
# Test LaunchServices V2 stdin forwarding with MCP server

set -e

echo "Testing MCP Screen Capture Server with LaunchServices V2..."
echo ""

# Create a test JSON-RPC request
TEST_REQUEST='{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":1}'

echo "Test Request:"
echo "$TEST_REQUEST"
echo ""

# Test with timeout to prevent hanging
echo "Sending initialize request to server..."
echo "$TEST_REQUEST" | timeout 10s ./mcp-screencapture-server 2>&1 || {
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 124 ]; then
        echo ""
        echo "⚠️  Server timed out (10s) - may indicate stdin forwarding issue"
        exit 1
    elif [ $EXIT_CODE -eq 0 ]; then
        echo ""
        echo "✅ Server responded and exited cleanly"
    else
        echo ""
        echo "❌ Server exited with code $EXIT_CODE"
        exit $EXIT_CODE
    fi
}

echo ""
echo "Test completed successfully!"
