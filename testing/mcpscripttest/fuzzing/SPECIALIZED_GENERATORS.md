# Specialized Generators for MCPScriptTest Fuzzing

Based on the analysis of generated scripts and your suggestions, I've created specialized generators that provide more control over test generation.

## Key Features

### 1. Command-Aware Generation
- Generators understand the available commands from the schema
- Can disable specific commands (like `exec` for safety)
- Weighted command selection for focused testing

### 2. Specialized Generator Types

#### MCPTraceGenerator
- Focuses on MCP protocol testing
- Disables `exec` commands for pure protocol testing
- Weighted towards MCP commands:
  - `mcp-trace`: 5.0
  - `mcp-send`: 3.0
  - `mcp-recv`: 3.0
  - `mcp-serve`: 2.0

```go
generator := fuzzing.NewMCPTraceGenerator(seed)
script := generator.Generate()
```

#### SafeFileOperationsGenerator 
- No `exec` or `rm` commands
- Focuses on safe file operations
- Weighted towards:
  - `cat`: 2.0
  - `cp`: 1.5
  - `mkdir`: 1.5
  - `stdout`: 2.0

```go
generator := fuzzing.NewSafeFileOperationsGenerator(seed)
script := generator.Generate()
```

### 3. Custom Configuration
Create generators with specific constraints:

```go
config := fuzzing.GeneratorConfig{
    DisabledCommands: map[string]bool{
        "exec": true,
        "rm":   true,
    },
    CommandWeights: map[string]float64{
        "mcp-send": 3.0,
        "stdin":    2.0,
    },
    AllowDirectives: false,
    MinScriptLength: 5,
    MaxScriptLength: 10,
}

generator := fuzzing.NewSpecializedGenerator(seed, config)
```

## Analysis of Current Generation Patterns

From analyzing the fuzzer output, I noticed:

1. **Heavy use of exec commands**: The original fuzzer frequently generates `exec` commands
2. **Mixed command types**: Scripts combine file ops, MCP commands, and system commands
3. **Platform directives**: Uses `[linux]`, `[!windows]`, etc.
4. **Weighted randomness**: Some commands appear more frequently

## Benefits of Specialized Generators

1. **Safety**: Can disable dangerous operations like `exec` and `rm`
2. **Focus**: Can target specific areas (MCP protocol, file operations)
3. **Control**: Fine-grained control over command weights
4. **Compliance**: Generate scripts that follow specific constraints

## Future Enhancements

1. **Schema-driven generation**: Use the actual command schema for validation
2. **Context-aware generation**: Generate commands that make sense in sequence
3. **State tracking**: Remember what files/resources exist for valid operations
4. **Grammar-based generation**: Use formal grammar for script generation

## Example Usage in Fuzzing

```go
func FuzzMCPProtocol(f *testing.F) {
    f.Add(int64(42))
    
    f.Fuzz(func(t *testing.T, seed int64) {
        // Use specialized generator for MCP testing
        generator := fuzzing.NewMCPTraceGenerator(seed)
        script := generator.Generate()
        
        // Run the test
        testScript(t, script)
    })
}
```

This approach allows for more targeted and safer fuzzing while maintaining the benefits of randomized testing.