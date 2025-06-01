# MCPSpec LSP

An experimental Language Server Protocol (LSP) implementation for MCP (Model Context Protocol) files:

- **mcptrace files** - `.mcp`, `.trace`, `.mcptrace` files containing MCP protocol traces
- **mcpscripttest files** - Script test files for testing MCP implementations

## Features

- Syntax validation for MCP trace files and scripttest files
- Code completion for common patterns
- Hover information for commands and keywords
- Diagnostics for common errors

## Installation

```bash
go install github.com/tmc/mcp/exp/cmd/mcpspec-lsp@latest
```

## Usage

The server implements the Language Server Protocol and can be used with any LSP-compatible editor:

### VS Code

1. Install the [vscode-languageserver-node](https://github.com/microsoft/vscode-languageserver-node) extension
2. Configure the extension to use `mcpspec-lsp` as the server command

Example configuration for VS Code settings.json:

```json
{
  "languageServer.mcpspec": {
    "command": "mcpspec-lsp",
    "args": ["-v"],
    "filetypes": [".mcp", ".trace", ".mcptrace", ".txt"]
  }
}
```

### Vim/Neovim with coc.nvim

1. Install [coc.nvim](https://github.com/neoclide/coc.nvim)
2. Add LSP configuration to `coc-settings.json`:

```json
{
  "languageserver": {
    "mcpspec": {
      "command": "mcpspec-lsp",
      "args": ["-v"],
      "filetypes": ["mcp", "trace", "mcptrace", "txt"]
    }
  }
}
```

## Development

This LSP server is experimental and in active development. Contributions are welcome!

### Building from source

```bash
cd /path/to/mcp/exp/cmd/mcpspec-lsp
go build
```

### Running with logging enabled

```bash
mcpspec-lsp -v -log /path/to/logfile.log
```

## Supported File Types

1. **MCP Trace Files** (`.mcp`, `.trace`, `.mcptrace`)
   - Records of MCP protocol communication
   - Format: `mcp-send {JSON}` or `mcp-recv {JSON}` with optional timestamps

2. **MCP Script Test Files** (`.txt` files in test directories)
   - Test scripts for MCP implementations
   - Format: Command lines starting with `>` followed by assertions

## License

Same as the MCP project. 