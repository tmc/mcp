# mcp-sort

Sort MCP trace files by timestamp or strip timestamps for comparison.

## Overview

`mcp-sort` is a utility for organizing and normalizing MCP trace files. It can:
- Sort entries by timestamp
- Strip timestamps entirely for content comparison
- Process files in-place
- Handle malformed entries gracefully

## Usage

```bash
mcp-sort [options] file1.mcp [file2.mcp ...]
```

## Options

- `-strip` - Strip timestamps instead of sorting
- `-output <file>` - Output file (default: stdout)
- `-in-place` - Edit files in place

## Examples

### Sort by Timestamp

Sort a trace file chronologically:
```bash
mcp-sort trace.mcp > sorted-trace.mcp
```

Sort multiple files:
```bash
mcp-sort file1.mcp file2.mcp file3.mcp > combined-sorted.mcp
```

### Strip Timestamps

Remove timestamps for content comparison:
```bash
mcp-sort -strip trace.mcp > trace-no-timestamps.mcp
```

### In-Place Editing

Sort file in place:
```bash
mcp-sort -in-place trace.mcp
```

### Pipeline Usage

Use in pipelines:
```bash
cat trace.mcp | mcp-sort > sorted.mcp
mcp-replay trace.mcp | mcp-sort -strip > normalized.mcp
```

## Use Cases

### 1. Pre-Diff Normalization

Prepare traces for comparison:
```bash
# Sort and strip timestamps before diffing
mcp-sort -strip trace1.mcp > trace1-norm.mcp
mcp-sort -strip trace2.mcp > trace2-norm.mcp
diff trace1-norm.mcp trace2-norm.mcp
```

### 2. Merge Multiple Traces

Combine and sort multiple recording sessions:
```bash
# Merge multiple trace files chronologically
mcp-sort session1.mcp session2.mcp session3.mcp > combined.mcp
```

### 3. Clean Trace Files

Fix out-of-order entries:
```bash
# Sort to fix chronological issues
mcp-sort messy-trace.mcp > clean-trace.mcp
```

### 4. Content-Only Comparison

Compare message content without timing:
```bash
# Strip timestamps for pure content comparison
mcp-sort -strip prod.mcp > prod-content.mcp
mcp-sort -strip test.mcp > test-content.mcp
mcpdiff prod-content.mcp test-content.mcp
```

## Timestamp Format

Recognizes timestamps in format:
```
[2024-05-10 14:32:15.123]
```

Or Unix timestamps:
```
# 1234567890.123
```

## Integration Examples

### With mcpdiff

Normalize before comparison:
```bash
# Sort both files first
mcp-sort trace1.mcp > trace1-sorted.mcp
mcp-sort trace2.mcp > trace2-sorted.mcp
mcpdiff trace1-sorted.mcp trace2-sorted.mcp
```

### With mcp-replay

Sort before replaying:
```bash
# Ensure chronological order for replay
mcp-sort raw-trace.mcp | mcp-replay -speed 2.0
```

### In Test Scripts

```bash
#!/bin/bash
# Normalize traces for testing
for trace in *.mcp; do
  mcp-sort -strip "$trace" > "normalized/${trace%.mcp}-norm.mcp"
done
```

## Performance

- Streaming processing for large files
- Efficient in-memory sorting
- Minimal memory footprint for strip operation

## Error Handling

- Skips malformed lines with warnings
- Preserves valid entries
- Returns non-zero exit code on errors

## Best Practices

1. **Always sort before replay** if timestamps might be out of order
2. **Strip timestamps** when comparing content only
3. **Use in-place** for large files to save disk space
4. **Combine with other tools** in pipelines

## See Also

- [mcp-replay](./mcp-replay.md) - Replay sorted traces
- [mcpdiff](./mcpdiff.md) - Compare normalized traces
- [mcp-spy](./mcp-spy.md) - Create trace files