# Using txtar Format in MCP Script Tests

The MCP scripttest framework leverages the txtar format from `golang.org/x/tools/txtar` to include test files directly within script test files. This approach eliminates the need for creating files using shell commands and makes tests more self-contained.

## What is txtar Format?

The txtar format combines script commands and data files within a single text file. It follows this structure:

```
# Script commands and test logic
command1
command2
...

-- filename1 --
Content for filename1 goes here...

-- filename2 --
Content for filename2 goes here...
```

## Benefits of Using txtar

1. **Self-contained tests**: All test files are included within the script test
2. **Cleaner test files**: No need for shell commands to create/manage test files
3. **Better readability**: Test content is clearly separated from test logic
4. **Consistent file state**: Ensures files have predictable content
5. **Language-agnostic**: Works with any file format or content

## Example: Testing mcpdiff with txtar

Here's an example of how to use txtar format to test the `mcpdiff` command:

```
# Test with default settings - should show differences in ID
mcpdiff sample1.mcp sample2.mcp
! stdout "Files match exactly!"
stdout "id"

# Test with -i flag to ignore IDs
mcpdiff -i sample1.mcp sample2.mcp
stdout "Files match exactly!"

-- sample1.mcp --
mcp-recv {"jsonrpc":"2.0","id":1,"method":"test"} # 1234.567
mcp-send {"jsonrpc":"2.0","id":1,"result":{"name":"test-server"}} # 1234.568

-- sample2.mcp --
mcp-recv {"jsonrpc":"2.0","id":1,"method":"test"} # 1234.567
mcp-send {"jsonrpc":"2.0","id":2,"result":{"name":"test-server"}} # 1234.568
```

## Example: Testing MCP Servers with txtar

When testing MCP servers, txtar is particularly useful for including sample MCP traces:

```
# Test replaying recorded trace
mcp-replay -f sample-trace.0.mcp -mock-server
stdout "Replaying 2 MCP messages from sample-trace.0.mcp"

-- sample-trace.0.mcp --
mcp-recv {"jsonrpc":"2.0","id":1,"method":"initialize","params":{}} # 1234567890.123
mcp-send {"jsonrpc":"2.0","id":1,"result":{"name":"test-server"}} # 1234567890.124
```

## Best Practices

1. **Always use txtar format** instead of shell commands to create files
2. **Include meaningful file content** that tests specific functionality
3. **Keep test files small and focused** on what's being tested
4. **Use descriptive filenames** within the txtar section
5. **Document the purpose** of each test file with comments

## Common Mistakes to Avoid

- **Don't create files with shell commands**:
  ```
  # Avoid this:
  echo '{"id":1}' > sample.json
  
  # Use txtar instead:
  -- sample.json --
  {"id":1}
  ```

- **Don't modify files with commands** after they're defined in txtar
- **Don't store large binary files** in txtar (use small files or mock responses)
- **Don't create files outside the test directory**

By following these guidelines, you'll create more robust, maintainable, and deterministic MCP script tests.