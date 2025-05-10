# ScriptTest Environment Guide

This document describes the environment available when running script-based tests using the `rsc.io/script/scripttest` framework with the MCP enhancements in the `github.com/tmc/mcp/exp/mcpscripttest` package.

## Overview

Script-based tests provide a powerful way to test CLI applications by simulating user interactions with the command line. The MCP project extends the `rsc.io/script/scripttest` framework to provide a controlled environment for testing MCP tools.

## Environment Structure

Each script test runs in a dedicated temporary directory with a controlled environment:

```
$WORK/
├── 001/          # Test execution directory
│   ├── tmp/      # Temporary directory for the test
│   └── ...       # Test files
```

### Environment Variables

The following environment variables are set in the scripttest environment:

| Variable | Description |
|----------|-------------|
| `USER`   | Current user name |
| `HOME`   | Temporary home directory to avoid interference from user configuration |
| `PATH`   | Minimal PATH that includes only the test binary directory |
| `PWD`    | Current working directory for the test |
| `WORK`   | Root working directory for the test |
| `TMPDIR` | Temporary directory for the test |

## Available Commands

The scripttest environment provides a rich set of commands for testing:

### File Operations

| Command | Description |
|---------|-------------|
| `cat files...` | Concatenate files and print to stdout |
| `cp src... dst` | Copy files to a target file or directory |
| `mkdir path...` | Create directories |
| `mv old new` | Rename a file or directory |
| `rm path...` | Remove a file or directory |
| `chmod perm paths...` | Change file mode bits |
| `symlink path -> target` | Create a symlink |

### Environment Operations

| Command | Description |
|---------|-------------|
| `cd dir` | Change the working directory |
| `env [key[=value]...]` | Set or log environment variables |
| `echo string...` | Display a line of text |

### Testing Utilities

| Command | Description |
|---------|-------------|
| `cmp [-q] file1 file2` | Compare files for differences |
| `cmpenv [-q] file1 file2` | Compare files with environment expansion |
| `exists [-readonly] [-exec] file...` | Check that files exist |
| `grep [-count=N] [-q] 'pattern' file` | Find lines matching a pattern |
| `stderr [-count=N] [-q] 'pattern'` | Find lines in stderr that match a pattern |
| `stdout pattern` | Verify stdout contains text |
| `skip [msg]` | Skip the current test |
| `stop [msg]` | Stop execution of the script |

### Process Control

| Command | Description |
|---------|-------------|
| `exec program [args...] [&]` | Run an executable program with arguments |
| `sleep duration [&]` | Sleep for a specified duration |
| `wait` | Wait for completion of background commands |

### MCP-Specific Commands

| Command | Description |
|---------|-------------|
| `mcp-replay recording [flags]` | Replay MCP recordings |
| `mcp-spy [flags]` | Spy on MCP traffic |
| `mcp-start [flags] [&]` | Start MCP components |
| `mcp-test [flags]` | Run MCP tests |
| `mcp-verify recording [flags]` | Verify MCP recordings |
| `mcp-send message [flags]` | Send MCP messages |
| `mcp-recv [flags]` | Receive MCP messages |
| `mcpdiff file1 file2 [flags]` | Compare MCP files |
| `mcpspy [flags]` | Spy on MCP traffic |

## Available Conditions

Conditions can be used with `[condition]` syntax to conditionally execute test blocks:

| Condition | Description |
|-----------|-------------|
| `[GOARCH:*]` | Matches if `runtime.GOARCH == <suffix>` |
| `[GOOS:*]` | Matches if `runtime.GOOS == <suffix>` |
| `[compiler:*]` | Matches if `runtime.Compiler == <suffix>` |
| `[exec:*]` | Matches if `<suffix>` names an executable in the PATH |
| `[root]` | Matches if running as root (os.Geteuid() == 0) |
| `[short]` | Matches if testing.Short() is true |
| `[verbose]` | Matches if testing.Verbose() is true |

## Creating Test Scripts

Script tests are text files with a `.txt` extension, typically stored in a `testdata/scripts/` directory. Each test script consists of a series of commands, with comments (`#`) and blank lines for readability.

### Basic Example

```
# Test a simple command
mcp-spy --help
stderr 'Usage:'

# Verify file operations
! exists testfile.txt
>testfile.txt sample content
exists testfile.txt
cat testfile.txt
stdout 'sample content'
```

### Command Prefixes

Commands can have these prefixes:

- `!` - Expect command to fail (non-zero exit code)
- `>` - Write content to a file (e.g., `>file.txt content`)
- `>>` - Append content to a file (e.g., `>>file.txt more content`)

## Controlling the Test Environment

The MCP project uses a custom `TestMain` function to create a highly controlled environment for scripttest:

```go
func TestMain(m *testing.M) {
    // Get the current working directory
    pwd, err := os.Getwd()
    if err != nil {
        fmt.Printf("Error getting working directory: %v\n", err)
        os.Exit(1)
    }

    // Keep track of what to clean up
    var cleanupFiles []string

    // Create a controlled test environment
    testEnv := setupTestEnvironment(pwd, &cleanupFiles)
    defer cleanupEnvironment(testEnv, cleanupFiles)

    // Run the tests
    code := m.Run()
    os.Exit(code)
}
```

The `setupTestEnvironment` function:

1. Builds the test binary in the current directory
2. Sets a minimal PATH that only includes the test binary
3. Creates a dummy HOME directory to avoid interference from user configuration
4. Sets a controlled temporary directory

## Running Script Tests

Script tests are executed using the `mcpscripttest.Test` function:

```go
func TestMyCommand(t *testing.T) {
    // Run all scripttest files in the testdata/scripts directory
    mcpscripttest.Test(t, "testdata/scripts/*.txt")
}
```

## Best Practices

1. **Keep tests focused**: Each script test should test a specific feature or behavior.
2. **Use clear comments**: Add comments to explain what each section of the test is doing.
3. **Clean up after tests**: Remove any temporary files or directories created during tests.
4. **Test error conditions**: Use the `!` prefix to test commands that should fail.
5. **Test with different inputs**: Test edge cases and boundary conditions.
6. **Test with realistic data**: Use real-world examples in your tests.

## Debugging Script Tests

To see detailed output from script tests, run the tests with the verbose flag:

```
go test -v ./cmd/mycommand
```

This will show the full execution trace of each script test, including the commands run, their output, and any errors encountered.

## See Also

- [Script Test Overview](./scripttest-overview.md)
- [Script Test Examples](./scripttest-examples.md)
- [MCP Test Framework](../development/testing.md)