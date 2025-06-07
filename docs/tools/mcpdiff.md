# mcpdiff

Compare MCP trace files and highlight differences with intelligent matching.

## Overview

`mcpdiff` compares two MCP trace files line by line, highlighting differences in protocol interactions. It supports semantic JSON comparison, timestamp normalization, and flexible matching options.

## Usage

```bash
mcpdiff [options] file1.mcp file2.mcp
```

## Options

### Comparison Options
- `-t` - Ignore timestamps when comparing (default: true)
- `-i` - Ignore IDs when comparing
- `-s` - Compare JSON values semantically (ignoring formatting)
- `-o` - Ignore order of messages (best effort matching)

### Output Options
- `-c <n>` - Number of context lines to show (default: 3)
- `-d` - Only show lines that differ
- `-v` - Verbose output (show all lines being considered)
- `-json` - Output differences in JSON format
- `-word-diff` - Show word-level differences
- `-no-color` - Disable colorized output

## Examples

### Basic Comparison

Compare two trace files:
```bash
mcpdiff trace1.mcp trace2.mcp
```

### Semantic JSON Comparison

Compare JSON content semantically:
```bash
mcpdiff -s server1.mcp server2.mcp
```

### Ignore Timestamps and IDs

Focus on protocol behavior only:
```bash
mcpdiff -t -i reference.mcp actual.mcp
```

### Show Only Differences

Compact output showing only changes:
```bash
mcpdiff -d trace1.mcp trace2.mcp
```

### Word-Level Differences

Show precise differences within lines:
```bash
mcpdiff -word-diff old.mcp new.mcp
```

## Output Format

### Standard Output

Differences are shown with color coding:
- 🔴 Red: Lines only in first file
- 🟢 Green: Lines only in second file
- 🟡 Yellow: Modified lines
- ⚪ Gray: Context lines

Example output:
```
@@ Line 5-8 @@
  mcp-send {"jsonrpc":"2.0","method":"initialize","id":1}
- mcp-recv {"jsonrpc":"2.0","result":{"version":"1.0"},"id":1}
+ mcp-recv {"jsonrpc":"2.0","result":{"version":"1.1"},"id":1}
  mcp-send {"jsonrpc":"2.0","method":"test","id":2}
```

### JSON Output

With `-json` flag:
```json
{
  "differences": [
    {
      "line": 6,
      "type": "modified",
      "file1": "mcp-recv {\"jsonrpc\":\"2.0\",\"result\":{\"version\":\"1.0\"},\"id\":1}",
      "file2": "mcp-recv {\"jsonrpc\":\"2.0\",\"result\":{\"version\":\"1.1\"},\"id\":1}"
    }
  ],
  "summary": {
    "total_differences": 1,
    "added": 0,
    "removed": 0,
    "modified": 1
  }
}
```

## Advanced Features

### Semantic Comparison

The `-s` flag enables semantic JSON comparison:
- Ignores whitespace differences
- Ignores key ordering
- Compares numeric values properly
- Handles nested structures

### Order-Independent Matching

With `-o` flag, mcpdiff attempts to match messages regardless of order:
```bash
mcpdiff -o unordered1.mcp unordered2.mcp
```

### Custom Context

Adjust context lines for better readability:
```bash
mcpdiff -c 5 trace1.mcp trace2.mcp  # Show 5 lines of context
mcpdiff -c 0 trace1.mcp trace2.mcp  # No context, only differences
```

## Use Cases

### Testing Server Implementations

Compare reference implementation with new server:
```bash
# Record reference behavior
mcp-spy -f reference.mcp -- ./reference-server < test-inputs.json

# Record new implementation
mcp-spy -f new.mcp -- ./new-server < test-inputs.json

# Compare
mcpdiff -s -i reference.mcp new.mcp
```

### Debugging Protocol Issues

Find where interactions diverge:
```bash
mcpdiff -v -word-diff working.mcp broken.mcp
```

### Regression Testing

Ensure behavior hasn't changed:
```bash
mcpdiff -t -i baseline.mcp current.mcp || exit 1
```

### Performance Analysis

Compare timing characteristics:
```bash
# Don't ignore timestamps
mcpdiff -t=false slow.mcp fast.mcp
```

## Integration Examples

### With mcp-spy

Record and compare:
```bash
# Record two runs
mcp-spy -f run1.mcp -- ./server --config=a
mcp-spy -f run2.mcp -- ./server --config=b

# Compare behavior
mcpdiff -s run1.mcp run2.mcp
```

### With mcp-replay

Normalize and compare:
```bash
# Strip timestamps before comparing
mcp-replay -strip trace1.mcp > normalized1.mcp
mcp-replay -strip trace2.mcp > normalized2.mcp
mcpdiff normalized1.mcp normalized2.mcp
```

### In CI/CD Pipelines

```bash
#!/bin/bash
# regression_test.sh

# Run test and capture trace
mcp-spy -f current.mcp -- make test

# Compare with baseline
if ! mcpdiff -s -i baseline.mcp current.mcp > diff.log; then
  echo "Regression detected:"
  cat diff.log
  exit 1
fi
```

## Best Practices

1. **Use semantic comparison (`-s`)** for functional testing
2. **Ignore timestamps (`-t`)** unless timing is important
3. **Ignore IDs (`-i`)** when IDs are auto-generated
4. **Save JSON output** for automated analysis
5. **Use word-diff** for detailed debugging

## Error Handling

### File Not Found
```bash
mcpdiff: error: file not found: missing.mcp
```

### Malformed Trace Files
```bash
mcpdiff: error: invalid MCP trace format at line 42
```

### Memory Issues
For very large files, consider:
```bash
# Process in chunks
split -l 10000 large.mcp part-
for f in part-*; do mcpdiff reference.mcp $f; done
```

## Performance Tips

- Use `-d` to reduce output size
- Disable color (`-no-color`) for faster processing
- Use specific comparison flags to skip unnecessary checks

## Exit Codes

- `0` - Files are identical (considering options)
- `1` - Files differ
- `2` - Error occurred

## See Also

- [mcp-spy](./mcp-spy.md) - Record traces for comparison
- [mcp-replay](./mcp-replay.md) - Replay and normalize traces
- [mcp-sort](./mcp-sort.md) - Sort traces before comparison
- [Testing Guide](../testing/README.md) - Testing strategies