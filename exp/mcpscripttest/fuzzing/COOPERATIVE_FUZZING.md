# Cooperative Fuzzing with Test Binaries

This guide explains how to convert test binaries to support cooperative fuzzing with mcpscripttest.

## Overview

Cooperative fuzzing allows test binaries to participate in the fuzzing process by:
- Generating valid command lines based on fuzzing seeds
- Validating command-line arguments
- Reporting their capabilities to the fuzzer
- Providing optimal test inputs

## Converting a Test Binary

### 1. Import the Framework

```go
import "github.com/tmc/mcp/exp/mcpscripttest"
```

### 2. Define Your Configuration

```go
config := mcpscripttest.TestBinaryConfig{
    BinaryName: "your-binary-name",
    
    SupportedFlags: []mcpscripttest.FlagDefinition{
        {
            Name:        "--verbose",
            ShortName:   "-v",
            Type:        "bool",
            Description: "Enable verbose output",
        },
        {
            Name:        "--output",
            ShortName:   "-o",
            Type:        "string",
            Default:     "output.txt",
            Description: "Output file path",
        },
    },
    
    AcceptsStdin: true,
    
    RequiredArgs: []mcpscripttest.ArgDefinition{
        {
            Name:        "input",
            Description: "Input file or pattern",
        },
    },
    
    GenerateFunc: generateCommand,
    ValidateFunc: validateArgs,
    ExecuteFunc:  execute,
}
```

### 3. Implement Required Functions

```go
// Generate valid command lines from a seed
func generateCommand(seed int64) (string, error) {
    rng := rand.New(rand.NewSource(seed))
    
    parts := []string{"your-binary"}
    
    // Randomly add flags
    if rng.Float64() < 0.5 {
        parts = append(parts, "-v")
    }
    
    // Add required arguments
    inputs := []string{"test.txt", "data.log", "input.json"}
    parts = append(parts, inputs[rng.Intn(len(inputs))])
    
    return strings.Join(parts, " "), nil
}

// Validate command-line arguments
func validateArgs(args []string) error {
    // Parse and validate flags
    // Check required arguments
    // Return error if invalid
    return nil
}

// Normal execution function
func execute() error {
    // Your normal program logic
    flag.Parse()
    // ...
    return nil
}
```

### 4. Use TestMainWithFuzzing

```go
func main() {
    mcpscripttest.TestMainWithFuzzing(config)
}
```

## Operating Modes

The framework supports multiple modes controlled by environment variables:

### Normal Mode (default)
Regular program execution.

### Generate Mode
```bash
MCP_SCRIPTTEST_GENERATE=1 ./your-binary 12345
```
Generates a valid command line using the provided seed.

### Validate Mode
```bash
MCP_SCRIPTTEST_VALIDATE=1 ./your-binary -v input.txt
```
Validates the provided arguments, exits 0 if valid, 1 if invalid.

### Introspect Mode
```bash
MCP_SCRIPTTEST_INTROSPECT=1 ./your-binary
```
Outputs JSON describing the binary's capabilities.

## Complete Example

See `examples/test_echo/main.go` for a complete working example.

```go
package main

import (
    "flag"
    "fmt"
    "math/rand"
    "strings"
    
    "github.com/tmc/mcp/exp/mcpscripttest"
)

var (
    uppercase = flag.Bool("u", false, "Convert to uppercase")
    repeat    = flag.Int("n", 1, "Number of repetitions")
)

func main() {
    config := mcpscripttest.TestBinaryConfig{
        BinaryName: "test_echo",
        
        SupportedFlags: []mcpscripttest.FlagDefinition{
            {
                Name:        "--uppercase",
                ShortName:   "-u",
                Type:        "bool",
                Description: "Convert to uppercase",
            },
            {
                Name:        "--repeat",
                ShortName:   "-n",
                Type:        "int",
                Default:     1,
                Description: "Number of repetitions",
            },
        },
        
        RequiredArgs: []mcpscripttest.ArgDefinition{
            {
                Name:        "message",
                Description: "Message to echo",
            },
        },
        
        GenerateFunc: generateCommand,
        ValidateFunc: validateArgs,
        ExecuteFunc:  execute,
    }
    
    mcpscripttest.TestMainWithFuzzing(config)
}

func generateCommand(seed int64) (string, error) {
    rng := rand.New(rand.NewSource(seed))
    parts := []string{"test_echo"}
    
    if rng.Float64() < 0.3 {
        parts = append(parts, "-u")
    }
    
    if rng.Float64() < 0.3 {
        parts = append(parts, fmt.Sprintf("-n %d", rng.Intn(5)+1))
    }
    
    messages := []string{"hello", "world", "test", "fuzzing"}
    parts = append(parts, messages[rng.Intn(len(messages))])
    
    return strings.Join(parts, " "), nil
}

func validateArgs(args []string) error {
    if len(args) == 0 {
        return fmt.Errorf("message required")
    }
    return nil
}

func execute() error {
    flag.Parse()
    
    if flag.NArg() == 0 {
        return fmt.Errorf("no message provided")
    }
    
    message := strings.Join(flag.Args(), " ")
    if *uppercase {
        message = strings.ToUpper(message)
    }
    
    for i := 0; i < *repeat; i++ {
        fmt.Println(message)
    }
    
    return nil
}
```

## Integration with SmartGenerator

The SmartGenerator will automatically detect and use binaries that support cooperative fuzzing:

1. It introspects binaries to check for support
2. Uses the generate mode to create valid commands
3. Can validate generated commands
4. Falls back to traditional generation if needed

## Benefits

- More efficient fuzzing with valid commands
- Better test coverage with domain-specific inputs
- Reduced false positives from invalid commands
- Faster fuzzing cycles with pre-validated inputs

## Best Practices

1. Keep generation fast (< 100ms)
2. Provide diverse but valid outputs
3. Include edge cases in generation
4. Validate thoroughly but efficiently
5. Use appropriate randomness from the seed
6. Document supported flags and arguments