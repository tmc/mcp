# Testing Unix Pipelines with Script Testing

This document provides guidance on how to test Unix pipelines and I/O redirection using the `mcpscripttest` framework.

## Introduction to Pipeline Testing

Unix pipelines (`|`) and redirections (`>`, `<`) are fundamental to command-line operations, enabling the composition of tools for complex data processing. When testing command-line tools, it's important to verify their behavior within pipelines.

## Limitations of Direct Shell Syntax

The scripttest framework does not directly support Unix shell syntax for pipes (`|`) and redirections (`>`, `<`). This means you cannot write traditional shell pipelines directly in test scripts:

```
# This WON'T work in scripttest
cat file.txt | grep "pattern" > results.txt
```

## Workarounds for Pipeline Testing

### 1. Using Pipe Operator with `cat`

You can simulate pipelines by using the pipe operator (`|`) with `cat` to feed file content to your command:

```
# Pipe content from a file to a command
cat input.txt | yourcommand
stdout "expected output"
```

Example:
```
# Feed JSON data to a parser
cat data.json | jsonparser
stdout "parsed 3 records"
```

### 2. Creating and Redirecting to Files

Use built-in file creation commands and then read those files:

```
# Create a file with content
echo "content" > tempfile.txt

# Run your command on that file
yourcommand tempfile.txt
stdout "expected output"

# Check output file contents
cat result.txt
stdout "processed content"
```

### 3. Testing stdin Input

To test a tool that reads from stdin:

```
echo "input data" | yourtool
stdout "processed: input data"
```

## Real-World Examples

### JSON Processing Pipeline

```
# Create JSON input
echo '{"key": "value"}' > input.json

# Process with jq-like tool
cat input.json | jsontool
stdout "key: value"

# Alternative approach with file redirects
jsontool < input.json > output.txt
cat output.txt
stdout "key: value"
```

### Log File Processing

```
# Create a log file
echo "2023-01-01 ERROR Test error" > error.log
echo "2023-01-01 INFO Test info" >> error.log

# Filter and process logs
cat error.log | logfilter --level=ERROR
stdout "2023-01-01 ERROR Test error"
! stdout "INFO"
```

### MCP Message Stream Processing

```
# Test processing of MCP messages
cat mcp_trace.txt | mcp-processor
stdout "Processed 5 messages"
stdout "Found 2 initialize requests"
```

## Sample MCP Tool Pipeline Test

Here's a complete example testing an MCP trace processing pipeline:

```
# Create a trace file
cat > trace.mcp <<EOF
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000001
mcp-recv {"jsonrpc":"2.0","method":"exit"} # 1683000002
EOF

# Test processing the trace
cat trace.mcp | mcp-analyzer
stdout "3 total messages"
stdout "2 received messages"
stdout "1 sent message"
stdout "Methods: initialize, exit"

# Test filtering specific message types
cat trace.mcp | mcp-analyzer --filter=recv
stdout "2 received messages"
! stdout "sent message"

# Test JSON output format
cat trace.mcp | mcp-analyzer --format=json > analysis.json
cat analysis.json
stdout '"total": 3'
stdout '"received": 2'
stdout '"sent": 1'
```

## Best Practices

1. **Pre-create test files**: Instead of complex pipelines, create test files using `echo` or the txtar file definition format
2. **Use `cat` with pipe**: The `cat file | command` pattern is widely supported
3. **Verify file contents**: After running commands, use `cat` to verify the contents of output files
4. **Use temporary files**: Create temporary files for intermediate pipeline steps
5. **Handle platform differences**: Be aware that pipeline behavior might differ on Windows

## Limitations

When working with pipelines in scripttest, keep these limitations in mind:

1. **No shell features**: Advanced shell features like process substitution aren't available
2. **No background pipelines**: You can't easily test background processes in pipelines
3. **Environment variables**: Environment variable expansion in pipelines might behave differently
4. **Platform differences**: Pipeline behavior varies between Unix and Windows

## See Also

- [Script Test Environment Guide](./scripttest-environment.md)
- [Txtar Guide](./txtar-guide.md)
- [MCP Testing Roadmap](../development/MCP_TESTING_ROADMAP.md)