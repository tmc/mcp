#!/bin/bash
# Demo script showing Plan9-style MCP trace server integration

set -e

echo "=== MCP9P Trace Server Demo ==="
echo
echo "This demonstrates Plan9 concepts applied to MCP trace management:"
echo "- Everything is a file"
echo "- Control files for commands"
echo "- Clone files for resource creation"
echo "- Live event streams"
echo "- Session-based namespaces"
echo

# Create demo trace files
mkdir -p traces/session-demo-abc123
mkdir -p traces/session-debug-def456

# Create sample trace files
cat > traces/session-demo-abc123/stdio.mcp << 'EOF'
# mcptrace:v1
# session-id: abc123
# transport: stdio
# client: claude-desktop/0.5.2
# created: 2024-01-15T10:30:45.123Z
mcp-recv {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}} # 1705312245.123
mcp-send {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"demo-server","version":"1.0.0"}}} # 1705312245.156
mcp-recv {"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}} # 1705312245.200
mcp-send {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"echo","description":"Echo tool"}]}} # 1705312245.234
EOF

cat > traces/session-demo-abc123/sse-session-xyz789.mcp << 'EOF'
# mcptrace:v1
# session-id: abc123
# transport: sse
# sse-session-id: xyz789
# created: 2024-01-15T10:30:45.167Z
mcp-send {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":2,"progress":50,"total":100}} # 1705312245.300
mcp-send {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":2,"progress":100,"total":100}} # 1705312245.350
EOF

cat > traces/session-debug-def456/stdio.mcp << 'EOF'
# mcptrace:v1
# session-id: def456
# transport: stdio
# client: mcp-inspector/1.0.0
# created: 2024-01-15T11:00:00.000Z
mcp-recv {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}} # 1705314000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","serverInfo":{"name":"debug-server","version":"2.0.0"}}} # 1705314000.050
EOF

echo "Created demo trace files:"
echo "  traces/session-demo-abc123/stdio.mcp"
echo "  traces/session-demo-abc123/sse-session-xyz789.mcp"
echo "  traces/session-debug-def456/stdio.mcp"
echo

echo "=== Plan9-Style Filesystem Interface ==="
echo
echo "With mcp-traces running, you would access traces like files:"
echo

echo "# List all sessions"
echo "curl http://localhost:9001/sessions"
echo "  session-demo-abc123/"
echo "  session-debug-def456/"
echo

echo "# List files in a session"
echo "curl http://localhost:9001/sessions/session-demo-abc123/"
echo "  stdio.mcp"
echo "  sse-session-xyz789.mcp"
echo "  ctl"
echo "  events"
echo "  stats"
echo

echo "# Read a trace file (just like 'cat')"
echo "curl http://localhost:9001/sessions/session-demo-abc123/stdio.mcp"
echo "  # mcptrace:v1"
echo "  # session-id: abc123"
echo "  mcp-recv {...} # 1705312245.123"
echo "  mcp-send {...} # 1705312245.156"
echo "  ..."
echo

echo "# Control session (just like writing to control file)"
echo "echo 'activate' | curl -X POST -d @- http://localhost:9001/sessions/abc123/ctl"
echo "  session abc123 activated"
echo

echo "# Create new session (Plan9 clone file pattern)"
echo "curl http://localhost:9001/clone"
echo "  session-1704567890"
echo

echo "# Follow live events (like 'tail -f')"
echo "curl -N http://localhost:9001/sessions/abc123/events"
echo "  data: {\"type\":\"file_updated\",\"session\":\"abc123\",\"file\":\"stdio.mcp\"}"
echo "  data: {\"type\":\"file_created\",\"session\":\"abc123\",\"file\":\"websocket.mcp\"}"
echo "  ..."
echo

echo "=== With FUSE Mount (Future) ==="
echo
echo "# Mount the trace namespace as a real filesystem"
echo "sudo mcp-fs -mount /mnt/mcp-traces -ns http://localhost:9001"
echo
echo "# Now use regular Unix tools:"
echo "ls /mnt/mcp-traces/sessions/"
echo "cat /mnt/mcp-traces/sessions/abc123/stdio.mcp"
echo "tail -f /mnt/mcp-traces/sessions/abc123/events"
echo "echo 'clear' > /mnt/mcp-traces/sessions/abc123/ctl"
echo "cat /mnt/mcp-traces/clone  # creates new session"
echo

echo "=== Integration with Existing Tools ==="
echo
echo "# Convert trace to HTML through the namespace"
echo "curl http://localhost:9001/sessions/abc123/stdio.mcp | mcptrace-to-html -o trace.html -open"
echo
echo "# Replay traces from namespace"
echo "curl http://localhost:9001/sessions/abc123/stdio.mcp | mcp-replay"
echo
echo "# Fake server using namespace traces"
echo "curl http://localhost:9001/sessions/abc123/stdio.mcp | mcp-fake"
echo

echo "=== Union Mounts (Advanced Plan9 Feature) ==="
echo
echo "# Combine traces from multiple servers into one view"
echo "mcp-mount -type union /all-traces \\"
echo "  http://server1:9001/sessions \\"
echo "  http://server2:9001/sessions \\"
echo "  http://server3:9001/sessions"
echo
echo "# Now /all-traces shows traces from all servers"
echo "curl http://localhost:9000/all-traces/"
echo "  server1-session-abc/"
echo "  server2-session-def/"  
echo "  server3-session-ghi/"
echo

echo "=== Benefits of Plan9 Approach ==="
echo
echo "✓ Familiar: Standard filesystem metaphors"
echo "✓ Composable: Works with Unix tools (cat, tail, grep, etc.)"
echo "✓ Live: Real-time monitoring through event streams"
echo "✓ Organized: Automatic session-based grouping"
echo "✓ Network transparent: Access traces from anywhere"
echo "✓ Simple: HTTP + filesystem interface"
echo "✓ Elegant: Everything is a file, clean hierarchy"
echo

echo "Demo complete! The trace files have been created in ./traces/"
echo "Run the actual mcp-traces server to see this in action."