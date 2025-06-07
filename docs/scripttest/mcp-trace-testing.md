# MCP Trace Testing Guide

This document provides guidance on writing script tests for MCP trace files using the `mcpscripttest` framework.

## Test Structure

The test files in script directories use the [txtar format](https://pkg.go.dev/golang.org/x/tools/txtar), which combines test scripts and test data files in a single text file.

### Script Test Format

Each script test file is structured with:

1. A series of commands to execute
2. File content for test data files

Example structure:
```
# Command line
exec some-command

# Another command
stdout 'expected output'

# File content definitions (note the double-dash prefix)
-- somefile.txt --
This is the content of somefile.txt

-- anotherfile.json --
{
  "key": "value"
}
```

### Available Commands

- `exec [command]`: Execute a command and expect a zero exit code
- `! [command]`: Execute a command and expect a non-zero exit code
- `stdout [regexp]`: Check that stdout from the previous command matches the regexp
- `stderr [regexp]`: Check that stderr from the previous command matches the regexp
- `stop`: Stop execution and fail the test
- `exists [file]`: Assert that a file exists
- `! exists [file]`: Assert that a file does not exist

### Test Environment

Each test runs in a temporary directory with:
- Environment variables preserved from the parent process
- Working directory set to the temporary directory
- Test data files from the txtar section written to the temporary directory

## Example Test

```
# Test basic functionality
exec tool-command -q sample.mcp
stdout 'expected output'

-- sample.mcp --
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}} # 1683000001
```

## Best Practices

1. Keep test cases focused on a single feature or behavior
2. Use descriptive test file names and comments
3. Define test data files using the txtar format with the `-- filename --` syntax
4. Prefer `stdout` and `stderr` assertions over file output comparisons
5. Use `exec` for commands expected to succeed and `!` for commands expected to fail

Remember that script tests don't support input/output redirection with `>` or `|` directly. For these cases, use the txtar format to prepare test files and the `exec` command to run the operation.

## See Also

- [Script Test Environment Guide](./scripttest-environment.md)
- [Txtar Guide](./txtar-guide.md)
- [MCP Testing Roadmap](../development/MCP_TESTING_ROADMAP.md)