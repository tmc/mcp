#!/bin/bash
# Example showing the requested TCP flow with mcp-proxy

# Original flow request:
# export SPYCMD="npx @modelcontextprotocol/server-everything stdio"
# socat TCP-LISTEN:7000,fork,reuseaddr EXEC:"mcpspy -v -vv -- ${SPYCMD}"

# Equivalent with mcp-proxy:
echo "Starting MCP proxy on TCP port 7000 with mcpspy integration..."
echo "This is equivalent to:"
echo '  socat TCP-LISTEN:7000,fork,reuseaddr EXEC:"mcpspy -v -vv -- npx @modelcontextprotocol/server-everything stdio"'
echo ""

# Using mcp-proxy
./mcp-proxy \
  -transport tcp \
  -listen :7000 \
  -spy \
  -spy-v \
  -spy-vv \
  -v \
  -t \
  -- \
  npx @modelcontextprotocol/server-everything stdio

# Notes:
# - mcp-proxy listens on TCP port 7000 (like socat TCP-LISTEN:7000)
# - Each new connection spawns a new process (like socat fork)
# - The -spy flag wraps the command with mcpspy
# - -spy-v and -spy-vv are passed to mcpspy for verbose output
# - The command after -- is what gets executed (the MCP server)