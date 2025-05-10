# Adding Custom Commands to Script Tests

This document explains how to add custom commands to script tests in the MCP project, which can be useful for mocking external tools and creating controlled test environments.

## Using MCPScripttest Options

The MCPScripttest package provides an option to add custom commands to the script test engine:

```go
options := mcpscripttest.DefaultOptions()

// Add a custom command
options.CustomCommands["my-command"] = script.Command(
    script.CmdUsage{
        Summary: "My custom command",
        Args:    "[args...]",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        // Handle the command execution
        return func(s *script.State) (string, string, error) {
            return "output", "stderr", nil
        }, nil
    },
)

// Run script tests with custom options
mcpscripttest.Test(t, "testdata/scripts/*.txt", options)
```

## Example: Mocking External CLI Tools

This approach is particularly useful for mocking external CLI tools in tests. Here's an example of how we mock the Claude CLI:

```go
// Add a custom claude command
options.CustomCommands["claude"] = script.Command(
    script.CmdUsage{
        Summary: "Claude AI assistant CLI",
        Args:    "[flags] [prompt]",
    },
    func(s *script.State, args ...string) (script.WaitFunc, error) {
        // Handle version flag
        if len(args) > 0 && (args[0] == "--version" || args[0] == "-v") {
            return func(s *script.State) (string, string, error) {
                return "claude version 1.0.0 (mock for testing)\n", "", nil
            }, nil
        }
        
        // Handle capabilities 
        if len(args) > 0 && args[0] == "capabilities" {
            return func(s *script.State) (string, string, error) {
                return `{"models":["claude-3-opus-20240229","claude-3-sonnet-20240229"],"tools":["bash","read_file"]}`, "", nil
            }, nil
        }
        
        // Handle any other prompts
        prompt := strings.Join(args, " ")
        response := fmt.Sprintf("Claude mock implementation called with args: %s\nThis is a mock Claude for testing\n", prompt)
        
        return func(s *script.State) (string, string, error) {
            return response, "", nil
        }, nil
    },
)
```

## Using in Script Tests

Once you've added a custom command, you can use it directly in script tests without needing 'exec':

```
# Test calling Claude command in script tests
claude --version
stdout 'claude version'

claude capabilities
stdout 'models'
stdout 'tools'

claude "Hello World"
stdout 'Claude mock implementation'
```

## Benefits of This Approach

1. **Controlled Environment**: Tests run in a completely controlled environment without depending on external tools.

2. **Deterministic Behavior**: Mocked commands always behave the same way, making tests more reliable.

3. **Fast Execution**: No need to execute real commands, which might be slow or have dependencies.

4. **Simplified Testing**: No need to deal with PATH or installation issues for external tools.

5. **Pure Testing**: Tests can run in CI environments without requiring installation of tools.

## When to Use Custom Commands vs. Real Commands

- Use custom commands when you want to mock external tools, especially those with complex logic or dependencies.
- Use real commands when you're specifically testing the interaction with the actual executable.

## Limitations

- Custom commands can't handle all complexities of real commands, so you need to implement the specific behaviors your tests need.
- IO redirection (>, <, |) might not work with custom commands in the same way as with real commands.