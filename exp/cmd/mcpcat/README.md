# mcpcat

Colorizes MCP trace files with syntax highlighting for better readability.

## Features

- **Direction-based coloring**:
  - `mcp-recv`: Green (client→server)
  - `mcp-send`: Bright cyan (server→client, more readable than blue)
  - `mcp-send-shadow`: Grey (shadow server responses)

- **JSON syntax highlighting**:
  - Method names in bold
  - ID values with cycling colors
  - Result/params highlighting
  - Timestamps in grey

- **Shadow mode support**:
  - Shadow server responses (`mcp-send-shadow`) are displayed entirely in grey
  - Makes it easy to distinguish primary from shadow traffic
  - Useful for comparing server implementations

- **Smart color detection**:
  - Auto mode: Colors when output is to a terminal, plain text otherwise
  - Respects the `NO_COLOR` environment variable
  - Manual control with `-color` flag

## Usage

```bash
# Colorize from file (auto-detect terminal)
mcpcat trace.mcp

# Colorize from stdin
cat trace.mcp | mcpcat

# Force color modes
mcpcat -color=always trace.mcp    # Always use color
mcpcat -color=never trace.mcp     # Never use color
mcpcat -color=auto trace.mcp      # Default: color if terminal

# Disable coloring with environment variable
NO_COLOR=1 mcpcat trace.mcp

# Legacy flag (deprecated)
mcpcat -c=false trace.mcp
```

## Example

Given a trace with shadow responses:
```
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1000.000
mcp-send {"jsonrpc":"2.0","result":{"capabilities":{}},"id":1} # 1001.000
mcp-send-shadow {"jsonrpc":"2.0","result":{"capabilities":{"shadow":true}},"id":1} # 1001.100
```

The tool will display:
- `mcp-recv` in green with highlighted JSON fields
- `mcp-send` in blue with highlighted JSON fields
- `mcp-send-shadow` entirely in grey (prefix, content, and timestamp)

## Color Modes

The `-color` flag supports three modes:

1. **auto** (default): Automatically detects if output is to a terminal
   - Colors when stderr is a TTY
   - Plain text when piped or redirected
   - Disabled if `NO_COLOR` is set

2. **always**: Force color output
   - Always outputs ANSI color codes
   - Useful when piping to `less -R` or similar
   - Still respects `NO_COLOR`

3. **never**: Disable color output
   - Always outputs plain text
   - Equivalent to setting `NO_COLOR=1`

## Shadow Mode

When using `mcp-shadow` to compare server implementations, the trace contains both primary and shadow responses. mcpcat makes these easy to distinguish by rendering all shadow traffic in grey, helping you quickly identify differences between implementations.

## Standard Compliance

mcpcat follows the [NO_COLOR](https://no-color.org/) standard: when the `NO_COLOR` environment variable is present (regardless of its value), all color output is disabled. This allows users to avoid ANSI color codes in terminal output when color is not supported or desired.