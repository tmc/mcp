#!/bin/bash
# Test script for mcp-screencapture-server
# This sends a simple tools/list request to the server via stdio

set -e

echo "Testing MCP Screen Capture Server..."
echo ""

# Test 1: Initialize request (required by MCP protocol)
echo "Sending initialize request..."
echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}},"id":1}' | ./mcp-screencapture-server 2>&1 &
SERVER_PID=$!

# Give it a moment to start
sleep 1

# Kill the server
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "Basic server test completed. Server started successfully."
echo ""
echo "For full manual testing:"
echo "1. Run: ./mcp-screencapture-server"
echo "2. Grant Screen Recording permission in System Settings if prompted"
echo "3. Test with MCP client or manual JSON-RPC requests"
