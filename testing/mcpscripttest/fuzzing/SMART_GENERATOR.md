# Smart Generator with Binary Introspection

The smart generator enhances fuzzing by analyzing test binaries to understand their command-line interfaces and generate valid commands.

## Features

### Binary Introspection
- Automatically discovers binary capabilities
- Extracts flag information from help text
- Detects stdin acceptance
- Generates valid command lines based on binary interface

### Engine Integration
- Validates commands using test binary validation mode
- Falls back to heuristics when validation isn't supported
- Enables early termination for invalid flags
- Reduces wasted fuzzing cycles on malformed commands

## How It Works

### 1. Binary Analysis
```go
introspector := NewBinaryIntrospector()
info, err := introspector.IntrospectBinary("echo")

// info contains:
// - Supported flags
// - Help text
// - Valid usage examples
// - Stdin acceptance
```

### 2. Smart Command Generation
The generator uses introspection data to create valid commands:
```go
config := SmartGeneratorConfig{
    EnableIntrospection: true,
    CommonTestBinaries:  true,
}

generator := NewSmartGenerator(seed, config)
script := generator.Generate()
```

### 3. Validation Mode
Test binaries can implement validation mode by checking the environment variable:
```go
if os.Getenv("MCP_SCRIPTTEST_VALIDATE_ONLY") == "1" {
    // Parse flags but don't execute
    flag.Parse()
    
    // Validate flag combinations
    if err := validateFlags(); err != nil {
        os.Exit(1) // Invalid flags
    }
    
    os.Exit(0) // Valid flags
}
```

### 4. Engine Integration
The smart generator can validate commands before including them:
```go
generator := NewSmartGeneratorWithEngine(seed, config)
script := generator.GenerateWithValidation()
```

## Binary Introspection Process

1. **Help Text Analysis**
   - Tries common help flags: `--help`, `-h`, `help`
   - Parses output for usage patterns
   - Extracts flag descriptions

2. **Flag Parsing**
   - Identifies flag formats: `-f`, `--flag`, `-f value`
   - Determines flag types: bool, string, int
   - Records flag descriptions

3. **Capability Detection**
   - Tests stdin acceptance
   - Identifies required vs optional arguments
   - Discovers subcommands

## Example Usage

### Basic Smart Generator
```go
config := SmartGeneratorConfig{
    EnableIntrospection: true,
    CommonTestBinaries:  true,
}

generator := NewSmartGenerator(seed, config)
script := generator.Generate()
```

### MCP-Focused Smart Generator
```go
generator := NewMCPSmartGenerator(seed)
script := generator.Generate()
```

### With Validation
```go
config := SmartGeneratorConfig{
    EnableIntrospection: true,
    ValidateCommands:    true,
}

generator := NewSmartGeneratorWithEngine(seed, config)
script := generator.GenerateWithValidation()
```

## Command Validation

Commands can be validated in two ways:

1. **Binary Validation Mode**: Test binaries check their own flags
2. **Fallback Heuristics**: Check flags against help text

### Implementing Validation Mode

Test binaries can support validation by:

```go
func main() {
    // Check for validation mode
    if os.Getenv("MCP_SCRIPTTEST_VALIDATE_ONLY") == "1" {
        // Only validate flags
        if err := flag.Parse(); err != nil {
            os.Exit(1)
        }
        
        // Custom validation logic
        if !isValidFlagCombination() {
            os.Exit(1)
        }
        
        os.Exit(0) // Valid
    }
    
    // Normal execution
    // ...
}
```

## Benefits

1. **Higher Quality Tests**: Generated commands are more likely to be valid
2. **Efficient Fuzzing**: Less time wasted on malformed commands
3. **Better Coverage**: Tests actual command-line interfaces
4. **Adaptable**: Learns from binaries without hardcoding

## Future Enhancements

1. **Subcommand Support**: Parse and generate subcommands
2. **Dependency Analysis**: Understand which commands need setup
3. **State Tracking**: Generate commands based on previous outputs
4. **Learning Mode**: Improve generation based on successful runs

## Example Output

With smart generation, instead of:
```
exec cat nonexistent.txt
exec grep -z invalid file
```

You get:
```
exec cat test.txt
exec grep -E "pattern" file.txt
exec echo "test message"
```

The smart generator significantly improves the quality of generated test scripts while maintaining the benefits of randomized fuzzing.