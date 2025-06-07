# MCP Diff Proxy Server

A Model Context Protocol (MCP) server that provides diff functionality as a service, wrapping system diff tools and mcpdiff.

## Features

- Text-based diff generation
- File-based diff generation  
- MCP trace file diff using mcpdiff
- Unified diff format output
- Error handling for missing files

## Tools

### text_diff
Generate a unified diff between two text strings.

**Parameters:**
- `text1` (string): First text to compare
- `text2` (string): Second text to compare

### file_diff
Generate a unified diff between two files.

**Parameters:**
- `file1` (string): Path to first file
- `file2` (string): Path to second file

### mcp_diff
Generate a diff between two MCP trace files using mcpdiff.

**Parameters:**
- `file1` (string): Path to first MCP trace file
- `file2` (string): Path to second MCP trace file

## Usage

```bash
# Start the server
go run .
```

## Examples

### Text diff
```json
{
  "tool": "text_diff",
  "arguments": {
    "text1": "Hello world\nThis is line 2",
    "text2": "Hello world\nThis is line 2 modified"
  }
}
```

### File diff
```json
{
  "tool": "file_diff",
  "arguments": {
    "file1": "/path/to/file1.txt",
    "file2": "/path/to/file2.txt"
  }
}
```

### MCP trace diff
```json
{
  "tool": "mcp_diff",
  "arguments": {
    "file1": "/path/to/trace1.mcp",
    "file2": "/path/to/trace2.mcp"
  }
}
```

## Dependencies

- System `diff` command
- `mcpdiff` command (for MCP trace diffs)

## Note

This server acts as a proxy to system diff tools, making diff functionality available through the MCP protocol for use by AI assistants and other MCP clients.