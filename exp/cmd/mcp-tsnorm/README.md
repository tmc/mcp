# mcp-tsnorm

A utility for normalizing timestamps in MCP trace files.

## Overview

`mcp-tsnorm` is a command-line tool that reads MCP trace files and adjusts the timestamps to a normalized format. It can:

- Rebase timestamps to start from a specific offset
- Convert timestamps to use an absolute time base
- Preserve the original time intervals while shifting to a new base
- Handle and optionally preserve MCP trace format headers

## Usage

```
mcp-tsnorm [options] [input_file]
```

If no input file is specified, it reads from standard input. Output goes to standard output by default.

### Options

- `-o <file>`: Write output to the specified file (default: stdout)
- `-start <duration>`: Specify the start offset for relative timestamps (e.g., "0s", "1.5s", "500ms") (default: "0s")
- `-absolute <timestamp>`: Rebase to a specific absolute Unix timestamp in seconds with millisecond precision (overrides -start)
- `-v`: Enable verbose mode to see details about timestamp conversion
- `-preserve-header`: Keep the mcptrace header if present (default: true)

## Examples

### Normalize a file to start at time 0:

```bash
mcp-tsnorm -o normalized.mcp recording.mcp
```

### Set a specific start time (1 second offset):

```bash
mcp-tsnorm -start 1s -o offset.mcp recording.mcp
```

### Rebase to an absolute Unix timestamp:

```bash
mcp-tsnorm -absolute 1621436800 -o absolute.mcp recording.mcp
```

### Strip the header while normalizing:

```bash
mcp-tsnorm -preserve-header=false -o noheader.mcp recording.mcp
```

### Use in pipelines:

```bash
cat recording.mcp | mcp-tsnorm -start 5s | mcpspy -v -f recording_normalized.mcp -- my_command
```

## Timestamp Format

MCP trace files use timestamps in the format: 
```
mcp-[direction] [content] # [seconds].[milliseconds]
```

The tool preserves the relative time differences between traces while adjusting the starting point.