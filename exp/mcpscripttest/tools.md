# MCPScriptTest Tools

This document describes the additional tools available for MCPScriptTest.

## Bash Command

The `bash` command allows running arbitrary bash commands within test scripts.

### Usage

```
bash 'command'
```

### Examples

```
# Simple echo
bash 'echo "Hello from bash"'
stdout 'Hello from bash'

# Using pipes
bash 'echo "one\ntwo\nthree" | grep two'
stdout 'two'

# With stdin
setstdin 'input data'
bash 'cat'
stdout 'input data'
```

See [BASH_COMMAND.md](BASH_COMMAND.md) for complete documentation.

## Schema Dumper

The schema dumper tool provides a comprehensive reference of all available scripttest commands.

### Usage

```bash
# Output schema as text to stdout
mcpscripttest-schema

# Output schema as markdown
mcpscripttest-schema -format markdown

# Save schema to a file
mcpscripttest-schema -format markdown -output commands.md
```

### Features

- Lists all core scripttest commands (exec, stdin, stdout, etc.)
- Documents MCP-specific commands (mcp-send, mcp-recv, etc.)
- Includes directives and modifiers (!, ?, [condition])
- Provides usage examples for each command
- Available in text or markdown format

## Fuzzing Support

MCPScriptTest includes fuzzing integration to help find edge cases and bugs.

### Go Fuzzing

The package includes a `FuzzScriptTest` function that works with Go's built-in fuzzing:

```go
// Run fuzzing
go test -fuzz=FuzzScriptTest ./exp/mcpscripttest
```

### Fuzz Generator Tool

Generate random but valid scripttest scripts:

```bash
# Generate a single script
mcpscripttest-fuzz

# Generate with specific seed
mcpscripttest-fuzz -seed 12345

# Generate multiple scripts
mcpscripttest-fuzz -count 10 -output fuzz

# This creates fuzz.0, fuzz.1, ..., fuzz.9
```

### Features

- Generates syntactically valid scripttest scripts
- Weighted probability for different command types
- Ensures at least one exec command per script
- Supports reproducible generation with seeds
- Can generate multiple scripts at once

## Sandbox Security

The sandbox feature allows running tests with restricted stdlib functionality.

### Configuration

```go
config := &mcpscripttest.SandboxConfig{
    BlockNetwork:     true,
    BlockExec:        true,
    BlockFileSystem:  true,
    AllowedPaths:     []string{"/tmp/test"},
    BlockEnvironment: true,
    AllowedEnvVars:   []string{"HOME", "PATH"},
    OverlayDir:       "/tmp/sandbox-overlay",
}
```

### Features

- **Network Blocking**: Prevents all network operations by stubbing the `net` package
- **Exec Blocking**: Prevents execution of external commands by stubbing `os/exec`
- **Filesystem Restrictions**: Limits file operations to allowed paths only
- **Environment Filtering**: Controls which environment variables are accessible

### Usage

```go
// Generate overlay files
err := mcpscripttest.GenerateBuildOverlay(config)

// Get the build command with overlay
cmd := mcpscripttest.GenerateBuildCommand(config, "mytest.go")

// Run sandboxed (placeholder implementation)
err = mcpscripttest.RunSandboxed(t, "TestMyFunction", config)
```

### Go Build Integration

The sandbox uses Go's build overlay feature:

```bash
# Build with sandbox
go build -overlay=/tmp/sandbox-overlay/overlay.json -tags=sandbox myprogram.go

# The overlay.json maps standard library files to our stubs
```

### Security Benefits

1. **Network Isolation**: Tests cannot make network connections
2. **Command Injection Protection**: Tests cannot execute arbitrary commands
3. **Filesystem Sandboxing**: Tests can only access allowed directories
4. **Environment Variable Control**: Tests only see approved env vars

### Implementation Notes

- Uses Go build overlays to replace stdlib packages at compile time
- Generates stub implementations that return errors for blocked operations
- Allows whitelisting specific paths or environment variables
- Can be combined with other security measures like containers

## Integration Examples

### Using Schema in Tests

```go
func TestCommandDocumentation(t *testing.T) {
    schema := mcpscripttest.GetSchema()
    
    // Verify all commands have examples
    for _, cmd := range schema.CoreCommands {
        if len(cmd.Examples) == 0 {
            t.Errorf("Command %s has no examples", cmd.Name)
        }
    }
}
```

### Fuzzing in CI

```yaml
# GitHub Actions example
- name: Run Fuzzing
  run: |
    go test -fuzz=FuzzScriptTest -fuzztime=30s ./exp/mcpscripttest
```

### Sandbox in Production

```go
// Use sandbox for untrusted scripts
if untrusted {
    config := &mcpscripttest.SandboxConfig{
        BlockNetwork: true,
        BlockExec:    true,
        OverlayDir:   filepath.Join(os.TempDir(), "sandbox"),
    }
    
    if err := RunSandboxed(t, scriptPath, config); err != nil {
        t.Fatalf("Sandboxed execution failed: %v", err)
    }
} else {
    mcpscripttest.Run(t, scriptPath)
}
```

## Future Improvements

1. **Schema Validation**: Validate scripts against the schema before execution
2. **Fuzzing Coverage**: Add coverage-guided fuzzing support
3. **Sandbox Profiles**: Pre-defined security profiles (strict, moderate, permissive)
4. **Schema Extensions**: Support for custom commands and validation rules
5. **Visual Schema Explorer**: Web-based tool to browse and search commands