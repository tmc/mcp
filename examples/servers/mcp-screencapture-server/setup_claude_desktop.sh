#!/bin/bash
# Script to configure Claude Desktop to use the screen capture MCP server

set -e

SERVER_PATH="/Volumes/tmc/go/src/github.com/tmc/mcp/examples/servers/mcp-screencapture-server/mcp-screencapture-server"
CLAUDE_CONFIG_DIR="$HOME/Library/Application Support/Claude"
CLAUDE_CONFIG_FILE="$CLAUDE_CONFIG_DIR/claude_desktop_config.json"

echo "=== Screen Capture MCP Server Setup for Claude Desktop ==="
echo ""

# Check if server binary exists
if [ ! -f "$SERVER_PATH" ]; then
    echo "❌ Server binary not found at $SERVER_PATH"
    echo "Please build it first with: cd $(dirname $SERVER_PATH) && go build"
    exit 1
fi

echo "✅ Server binary found"

# Create Claude config directory if it doesn't exist
mkdir -p "$CLAUDE_CONFIG_DIR"

# Create or update configuration
if [ -f "$CLAUDE_CONFIG_FILE" ]; then
    echo "📝 Existing Claude Desktop configuration found"
    echo "Current configuration:"
    cat "$CLAUDE_CONFIG_FILE"
    echo ""
    echo "To add the screen capture server, merge this into your config:"
else
    echo "📝 Creating new Claude Desktop configuration"
fi

cat << EOF
{
  "mcpServers": {
    "screencapture": {
      "command": "$SERVER_PATH",
      "args": []
    }
  }
}
EOF

echo ""
echo "=== Next Steps ==="
echo "1. Copy the above configuration to: $CLAUDE_CONFIG_FILE"
echo "   (Merge with existing servers if you have any)"
echo ""
echo "2. Restart Claude Desktop"
echo ""
echo "3. On first use, macOS will prompt for Screen Recording permission"
echo "   - Open System Settings → Privacy & Security → Screen Recording"
echo "   - Enable 'ScreenCaptureMCP'"
echo ""
echo "4. Restart Claude Desktop again after granting permission"
echo ""
echo "=== Test the Server Manually ==="
echo "You can test the server locally first:"
echo "  $SERVER_PATH"
echo ""
echo "Then send JSON-RPC requests:"
echo '  echo '"'"'{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}},"id":1}'"'"' | $SERVER_PATH'
