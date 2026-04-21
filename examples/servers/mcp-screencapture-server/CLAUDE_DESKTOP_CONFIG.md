# Claude Desktop Configuration for Screen Capture MCP Server

Add this configuration to your Claude Desktop config file to use the screen capture server.

## Configuration File Location

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

## Configuration

```json
{
  "mcpServers": {
    "screencapture": {
      "command": "/Volumes/tmc/go/src/github.com/tmc/mcp/examples/servers/mcp-screencapture-server/mcp-screencapture-server",
      "args": []
    }
  }
}
```

If you already have other MCP servers configured, add the `screencapture` entry to your existing `mcpServers` object.

## First Run

1. **Save the configuration** to `claude_desktop_config.json`
2. **Restart Claude Desktop** to load the new server
3. **Grant Screen Recording permission** when prompted by macOS
   - The macgo framework will request Screen Recording permission
   - Open System Settings → Privacy & Security → Screen Recording
   - Enable permission for the ScreenCaptureMCP application
4. **Restart Claude Desktop again** after granting permission

## Available Tools

Once configured, you'll have access to:

- **`list_screens`**: Lists all connected displays and their specifications
- **`capture_screen`**: Captures a screenshot and returns it as an image

## Example Usage in Claude Desktop

Try asking Claude:

- "Can you show me my current screen?"
- "What displays do I have connected?"
- "Take a screenshot and analyze what's on my screen"

The server will use macgo to maintain a stable TCC identity for Screen Recording permissions.
