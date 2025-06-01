#!/bin/bash
# Demonstrates mcp-probe with server interaction
echo "Running mcp-probe with minimal_server..."
echo ""
./mcp-probe -v ./minimal_server
echo ""
echo "Note: Direct Unix piping (mcp-probe | server) is not supported"
echo "because MCP requires bidirectional communication."
echo ""
echo "Alternative approaches:"
echo "1. Use as subprocess: ./mcp-probe ./server" 
echo "2. Use with TCP: ./mcp-probe -http http://localhost:8080"
echo "3. Use with SSE: ./mcp-probe -sse http://localhost:8080/sse"