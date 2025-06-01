# MCP System Server

A Model Context Protocol (MCP) server that provides system information and basic system administration capabilities.

## ⚠️ Security Warning

This server includes system execution capabilities and should be used with caution. Only use in trusted environments and be careful when granting access to system commands.

## Features

- **System Information**: Get OS, hardware, and environment details
- **Process Management**: List and monitor running processes
- **Disk Usage**: Check disk space and filesystem information
- **Environment Variables**: Access and filter environment variables
- **Command Execution**: Execute system commands (use with caution)

## Setup

1. **Build the Server**:
   ```bash
   go build -o mcp-system-server
   ```

2. **Run the Server**:
   ```bash
   ./mcp-system-server
   ```

## Tools

### `get_system_info`
Get general system information including OS, hardware, and user details.

**Parameters:**
- `include_env` (optional): Whether to include environment variables (default: false)

**Example:**
```json
{
  "include_env": true
}
```

**Response:**
```json
{
  "os": "darwin",
  "architecture": "arm64",
  "num_cpu": 8,
  "go_version": "go1.21.0",
  "hostname": "MacBook-Pro.local",
  "username": "john",
  "home_dir": "/Users/john",
  "working_dir": "/Users/john/projects",
  "uptime": "5h30m15s",
  "environment": {
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "HOME": "/Users/john"
  }
}
```

### `list_processes`
List running processes with filtering and limiting options.

**Parameters:**
- `limit` (optional): Maximum number of processes to return (default: 20)
- `filter` (optional): Filter processes by name (case-insensitive substring match)

**Example:**
```json
{
  "limit": 10,
  "filter": "node"
}
```

**Response:**
```json
{
  "processes": [
    {
      "pid": 1234,
      "name": "node",
      "command": "node server.js",
      "cpu_usage": "2.5%",
      "memory_usage": "1.2%",
      "status": "R"
    }
  ],
  "count": 1
}
```

### `get_disk_usage`
Get disk usage information for a specified path.

**Parameters:**
- `path` (optional): Path to check disk usage for (default: current directory)

**Example:**
```json
{
  "path": "/Users/john"
}
```

**Response:**
```json
{
  "path": "/Users/john",
  "filesystem": "/dev/disk1s1",
  "size": "465Gi",
  "used": "123Gi", 
  "available": "341Gi",
  "use_percent": "27%",
  "mountpoint": "/"
}
```

### `get_env_vars`
Get environment variables with optional filtering.

**Parameters:**
- `filter` (optional): Filter environment variables by name (case-insensitive substring match)

**Example:**
```json
{
  "filter": "PATH"
}
```

**Response:**
```json
{
  "environment_variables": {
    "PATH": "/usr/local/bin:/usr/bin:/bin",
    "MANPATH": "/usr/local/share/man:/usr/share/man"
  },
  "count": 2
}
```

### `execute_command`
Execute a system command with optional arguments and timeout.

**⚠️ Use with extreme caution - this can execute any system command!**

**Parameters:**
- `command` (required): The command to execute
- `args` (optional): Array of command arguments
- `timeout` (optional): Timeout in seconds (default: 30)

**Example:**
```json
{
  "command": "ls",
  "args": ["-la", "/tmp"],
  "timeout": 10
}
```

**Response:**
```json
{
  "command": "ls",
  "args": ["-la", "/tmp"],
  "output": "total 8\ndrwxr-xr-x  5 root  wheel  160 Jan 15 10:30 .\n...",
  "duration": "15ms",
  "exit_code": 0
}
```

## Security Considerations

### Command Execution
- The `execute_command` tool can run any system command
- Always validate commands before execution
- Consider implementing a whitelist of allowed commands
- Set appropriate timeouts to prevent hanging processes

### Environment Variables
- May contain sensitive information (API keys, passwords)
- Use filtering to limit exposure of sensitive variables
- Consider excluding sensitive environment variables

### Process Information
- May reveal sensitive application details
- Process arguments might contain sensitive data

## Platform Support

### Supported Platforms
- **Linux**: Full support for all features
- **macOS**: Full support for all features
- **Windows**: Partial support (some commands may behave differently)

### Platform-Specific Behaviors
- **Process listing**: Uses `ps aux` on Unix-like systems, `tasklist` on Windows
- **Disk usage**: Uses `df -h` on Unix-like systems, `fsutil` on Windows
- **Uptime**: Reads `/proc/uptime` on Linux, uses `uptime` command on macOS

## Configuration for Claude Desktop

Add this to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "system": {
      "command": "/path/to/mcp-system-server"
    }
  }
}
```

## Example Usage

Once connected, you can ask Claude:
- "What's the system information for this machine?"
- "Show me all running processes"
- "What's the disk usage of my home directory?"
- "List all environment variables containing 'PATH'"
- "Execute 'uptime' command"
- "Show me processes running 'python'"

## Common Commands

### Safe System Information
```json
{
  "command": "uptime"
}
```

```json
{
  "command": "whoami"
}
```

```json
{
  "command": "pwd"
}
```

### File System Operations
```json
{
  "command": "ls",
  "args": ["-la"]
}
```

```json
{
  "command": "df",
  "args": ["-h"]
}
```

### Process and Resource Information
```json
{
  "command": "top",
  "args": ["-l", "1", "-n", "5"]
}
```

## Error Handling

The server includes comprehensive error handling for:
- Invalid command syntax
- Command execution failures
- Timeout errors
- Permission errors
- File system access errors
- Platform-specific command differences

All errors are returned in a structured format with descriptive messages.