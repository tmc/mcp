# MCP Filesystem Server Example

This example demonstrates implementing a simple MCP server that provides filesystem access.

## Usage

```bash
go run . /path/to/allowed/directory [additional-directories...]
```

The server will only allow access to files within the specified directories.

## Features

- Read file contents
- Basic security model (TODO)
- JSON-RPC over stdio

## Tools

- read_file: Read contents of a file
