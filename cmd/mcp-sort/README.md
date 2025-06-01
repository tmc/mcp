# mcp-sort

A utility for sorting MCP trace files by timestamp and stripping timestamps for diffing.

## Overview

`mcp-sort` is a command-line tool for working with MCP (Model Context Protocol) trace files. 
It provides two primary functions:

1. **Sorting**: Sort MCP trace entries chronologically by their timestamps
2. **Timestamp Stripping**: Replace actual timestamps with a placeholder (`[TIMESTAMP]`) for better diffing

## Installation

```bash
go install github.com/tmc/mcp/cmd/mcp-sort@latest
```

Or build from source:

```bash
go build -o mcp-sort .
```

## Usage

```
Usage: mcp-sort [options] [files...]

Reads MCP trace files and sorts the entries by timestamp.
If no files are specified, reads from stdin.

Options:
  -in-place
        Edit files in place
  -output string
        Output file (default: stdout)
  -strip
        Strip timestamps instead of sorting
```

## Examples

### Sorting MCP Trace Files

Sort a single file and print to stdout:
```bash
mcp-sort input.mcp
```

Sort multiple files:
```bash
mcp-sort file1.mcp file2.mcp
```

Sort and save to a new file:
```bash
mcp-sort -output=sorted.mcp input.mcp
```

Sort in-place (modifies original file):
```bash
mcp-sort -in-place file.mcp
```

### Stripping Timestamps

Replace all timestamps with `[TIMESTAMP]` placeholder:
```bash
mcp-sort -strip input.mcp
```

Strip timestamps and save to a new file:
```bash
mcp-sort -strip -output=stripped.mcp input.mcp
```

Strip timestamps in-place:
```bash
mcp-sort -strip -in-place file.mcp
```

### Using with Pipes

```bash
cat input.mcp | mcp-sort
```

```bash
cat input.mcp | mcp-sort -strip > stripped.mcp
```

## Integration with Vim

The MCP Vim plugin includes integration with mcp-sort for timestamp-insensitive diffing. 
See the Vim plugin documentation for more details.

## Notes

- The tool expects MCP trace lines starting with `mcp-send` or `mcp-recv` followed by a timestamp in the format `[YYYY-MM-DD HH:MM:SS.sss]`
- Non-conforming lines are preserved but not sorted
- Sorting is stable (maintains original order of entries with identical timestamps)