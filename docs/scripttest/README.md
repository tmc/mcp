# MCP Script Testing Documentation

This directory contains documentation for script-based testing in the MCP project.

## Overview

Script-based testing is a powerful approach for testing command-line tools. The MCP project uses the `rsc.io/script/scripttest` framework, enhanced with MCP-specific functionality to test MCP tools.

## Key Documents

- [**Script Test Environment Guide**](./scripttest-environment.md): Describes the environment available when running script tests, including available commands, conditions, and environment variables.

- [**Controlled Test Environment Guide**](./controlled-test-environment.md): Explains how to implement a controlled environment for script tests to ensure isolation and reproducibility.

- [**txtar Guide**](./txtar-guide.md): Guide to the txtar format used for storing script tests with inline test files.

- [**Script Test Blog Post**](./scripttest-blog-post-enhanced.md): Comprehensive blog post about script testing with enhanced guidance for MCP.

- [**Script Test LLM Guide**](./scripttest-llm-guide-enhanced.md): Detailed guide for using script tests, tailored for LLM usage.

## Getting Started

To create script tests for an MCP command:

1. Create a `testdata/scripts/` directory in your command package.

2. Create `.txt` files in that directory with script test commands. See the [Script Test Environment Guide](./scripttest-environment.md) for available commands.

3. Set up a controlled test environment using the `TestMain` function as described in the [Controlled Test Environment Guide](./controlled-test-environment.md).

4. Run your tests with `go test ./cmd/yourcommand`.

## Example

A simple script test file (`testdata/scripts/basic.txt`):

```
# Test help command
mycmd --help
stderr 'Usage:'

# Test version command
mycmd --version
stdout '1.0.0'

# Create a test file
>testfile.txt content
exists testfile.txt

# Test file processing
mycmd process testfile.txt
stdout 'Processed: content'
```

## Testing Tips

1. **Organize tests logically**: Group related commands together and use comments to explain the purpose of each section.

2. **Test error conditions**: Use the `!` prefix to test commands that should fail.

3. **Use the environment information**: Leverage the controlled environment to ensure tests are reproducible.

4. **Check both stdout and stderr**: Verify both standard output and standard error as appropriate.

5. **Clean up after your tests**: Remove any temporary files or directories created during tests.

## See Also

- [MCP Test Framework](../development/testing.md)
- [scripttest Package Documentation](https://pkg.go.dev/rsc.io/script/scripttest)
- [MCP ScriptTest Package](https://pkg.go.dev/github.com/tmc/mcp/exp/mcpscripttest)