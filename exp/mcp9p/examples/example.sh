#!/bin/bash
# MCP Namespace System Example

# Start the namespace server in the background
echo "Starting namespace server..."
mcp-namespace -addr :9000 &
NS_PID=$!
sleep 2

# Register some local services
echo "Registering local services..."
mcp-ns -s http://localhost:9000 -c register /services/echo \
  -type local \
  -transport stdio \
  -command "npx" \
  -args "@modelcontextprotocol/server-echo,stdio"

mcp-ns -s http://localhost:9000 -c register /services/time \
  -type local \
  -transport stdio \
  -command "npx" \
  -args "@modelcontextprotocol/server-time,stdio"

# Create a namespace for AI services
mcp-ns -s http://localhost:9000 -c register /services/ai \
  -type namespace

# Register a remote service
mcp-ns -s http://localhost:9000 -c register /services/ai/calculator \
  -type remote \
  -transport http \
  -address "http://calculator.example.com/mcp"

# List all services
echo -e "\nListing all services:"
mcp-ns -s http://localhost:9000 -c list /services

# Look up a specific service
echo -e "\nLooking up echo service:"
mcp-ns -s http://localhost:9000 -c lookup /services/echo

# Create a mount
echo -e "\nCreating mount /local/echo -> /services/echo:"
mcp-mount -ns localhost:9000 /services/echo /local/echo

# Auto-mount a new service
echo -e "\nAuto-mounting filesystem service:"
mcp-mount -ns localhost:9000 -type auto /services/fs -- \
  npx @modelcontextprotocol/server-filesystem stdio

# Use mcp-tunnel with namespace
echo -e "\nCreating tunnel for echo service:"
mcp-tunnel -ns-server http://localhost:9000 \
  -namespace ns://localhost:9000/public/echo \
  -- ns://localhost:9000/services/echo &
TUNNEL_PID=$!
sleep 3

# List services again to see the tunneled service
echo -e "\nListing services after tunnel creation:"
mcp-ns -s http://localhost:9000 -c list /public

# Mount namespace as filesystem (requires root/FUSE)
if command -v mcp-fs &> /dev/null && [ -d /tmp/mcp ]; then
  echo -e "\nMounting namespace as filesystem:"
  mkdir -p /tmp/mcp
  mcp-fs -mount /tmp/mcp -ns http://localhost:9000 &
  FS_PID=$!
  sleep 2
  
  echo "Filesystem contents:"
  find /tmp/mcp -type f | head -10
  
  # Clean up filesystem mount
  fusermount -u /tmp/mcp 2>/dev/null || umount /tmp/mcp 2>/dev/null
  kill $FS_PID 2>/dev/null
fi

# Clean up
echo -e "\nCleaning up..."
kill $TUNNEL_PID 2>/dev/null
kill $NS_PID 2>/dev/null

echo "Example completed!"