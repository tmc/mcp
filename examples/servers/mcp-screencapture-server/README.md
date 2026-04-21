# Screen Capture MCP Server

A Model Context Protocol (MCP) server that provides screen capture capabilities for macOS.

This server demonstrates:
1.  **Official Go MCP SDK Integration**: Uses `github.com/tmc/mcp`.
2.  **Macgo Integration**: Uses `macgo` to bundle the app and manage permissions (Screen Recording).
3.  **Stdio Transport**: Enables usage with any MCP client (Claude Desktop, etc.).

## Tools

*   `list_screens`: Lists connected displays using `system_profiler SPDisplaysDataType`.
*   `capture_screen`: Captures a screenshot and returns it as a PNG image.

## Usage

### 1. Build

```bash
go build -o mcp-screencapture-server
```

### 2. Run / Permission Check

Run locally first to trigger the macOS Screen Recording permission prompt (TCC).

```bash
./mcp-screencapture-server
```

> **Note**: The first run might look like it hangs while waiting for you to grant permission in System Settings. You may need to restart the terminal or app after granting permission.

### 3. Use with MCP Client

Configure your MCP client (e.g., `claude_desktop_config.json`) to run the binary:

```json
{
  "mcpServers": {
    "screencapture": {
      "command": "/absolute/path/to/mcp-screencapture-server",
      "args": []
    }
  }
}
```

### 4. Testing

Run the unit tests:

```bash
go test -v .
```

## How It Works

This server runs over `stdio` using macgo's LaunchServices V2:

1.  **LaunchServices Launch**: The server uses macgo's LaunchServices to create an app bundle on-the-fly
2.  **TCC Identity**: The app bundle provides a stable identity for Screen Recording permissions
3.  **Stdin/Stdout Forwarding**: LaunchServices V2 automatically forwards stdin/stdout/stderr between parent and child processes
4.  **MCP Communication**: JSON-RPC messages flow through the forwarded stdio streams

This approach provides both:
- Proper TCC identity (via app bundle) for Screen Recording permission
- Preserved stdio streams for MCP JSON-RPC communication

No `ForceDirectExecution` is needed - LaunchServices V2 handles stdio forwarding automatically.

## macOS TCC Permissions

Screen recording requires macOS Screen Recording permission (TCC). When you first run the server:

1. macOS will prompt you to grant Screen Recording permission
2. Open System Settings → Privacy & Security → Screen Recording
3. Enable the permission for the application
4. Restart the server

The `macgo` library handles the TCC identity management automatically.
