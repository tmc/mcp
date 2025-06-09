# MCPScriptTest Fuzzing

This directory contains coverage-guided fuzzing capabilities for MCPScriptTest.

## Features

- **Coverage-Guided Fuzzing**: Uses Go's coverage data to guide fuzzing toward uncovered code paths
- **Script Generation**: Automatically generates valid scripttest scripts for fuzzing
- **Mutation-Based Input Generation**: Mutates successful test inputs to explore new code paths
- **Direct State-Based Fuzzing**: Run tests directly without file system overhead
- **Incremental Coverage Tracking**: Tracks and prioritizes inputs that increase code coverage
- **Live Visualization**: Real-time display of fuzzing progress and test scripts
- **Selective Output**: Only shows accepted scripts by default (configurable)

## Usage

### Basic Fuzzing

```go
// In your test file
func FuzzMyServer(f *testing.F) {
    serverCmd := []string{"go", "run", "./server"}
    fuzzing.FuzzWithState(f, serverCmd, nil)
}
```

### Coverage-Guided Fuzzing

```go
// Run with coverage collection enabled
// Set GOCOVERDIR environment variable to collect coverage data
GOCOVERDIR=/tmp/coverage go test -fuzz=FuzzMyServer
```

### Direct Fuzzing with Run()

```go
import "github.com/tmc/mcp/exp/mcpscripttest/fuzzing"

func TestWithFuzzing(t *testing.T) {
    opts := fuzzing.DefaultRunOptions()
    opts.Iterations = 10000
    opts.MinCoverage = 80.0
    opts.Verbose = true
    
    err := fuzzing.Run(func(script string) error {
        // Your test logic here
        return runScript(script)
    }, opts)
}
```

### Live Visualization

The fuzzer includes a real-time visualization feature that shows what scripts are being tested:

```go
// Enable visualization in Run()
vizOpts := fuzzing.DefaultVisualizerOptions()
vizOpts.Enabled = true
vizOpts.ShowRejected = false  // Only show accepted scripts
viz := fuzzing.NewVisualizer(vizOpts)

opts := fuzzing.DefaultRunOptions()
opts.Visualizer = viz

fuzzing.Run(testFunc, opts)
```

For fuzz tests, use environment variables:

```bash
# Enable visualization
MCP_FUZZ_VISUALIZE=1 go test -fuzz=FuzzMyServer

# Also show rejected scripts
MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_SHOW_REJECTED=1 go test -fuzz=FuzzMyServer

# Clear screen between updates
MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_CLEAR_SCREEN=1 go test -fuzz=FuzzMyServer
```

Visualization is automatically enabled with `-v` flag:
```bash
go test -v -fuzz=FuzzMyServer
```

The visualizer displays:
- Total scripts tested
- Acceptance/rejection rates
- Current script being tested (only shows accepted scripts by default)
- Coverage improvements when detected
- Final statistics

## Components

- **FuzzGenerator**: Generates valid scripttest scripts with weighted command selection
- **CoverageFeedback**: Collects and analyzes coverage data to guide fuzzing
- **CoverageGuidedFuzzer**: Combines generation and mutation strategies based on coverage
- **RunWithState**: Executes scripts directly with state management
- **Visualizer**: Live display of fuzzing progress and test scripts
- **SpecializedGenerators**: Targeted generators for specific test scenarios
- **SmartGenerator**: Uses binary introspection to generate valid commands
- **BinaryIntrospector**: Analyzes binaries to understand their interfaces
- **EngineValidator**: Validates commands using test binary cooperation

### Specialized Generators

The fuzzing package includes specialized generators for focused testing:

- **MCPTraceGenerator**: Focuses on MCP protocol without exec commands
- **SafeFileOperationsGenerator**: File operations without exec/rm
- **Custom Configurations**: Create generators with specific constraints

```go
// MCP-focused testing without exec
generator := fuzzing.NewMCPTraceGenerator(seed)

// Safe file operations
generator := fuzzing.NewSafeFileOperationsGenerator(seed)

// Custom configuration
config := fuzzing.GeneratorConfig{
    DisabledCommands: map[string]bool{"exec": true},
    CommandWeights: map[string]float64{"mcp-send": 3.0},
}
generator := fuzzing.NewSpecializedGenerator(seed, config)
```

See [SPECIALIZED_GENERATORS.md](SPECIALIZED_GENERATORS.md) for detailed documentation.

### Smart Generator

The smart generator analyzes test binaries to understand their interfaces:

- **Binary Introspection**: Discovers flags and options automatically
- **Validation Mode**: Cooperates with test binaries to validate commands
- **Intelligent Generation**: Creates valid command lines based on analysis

```go
// Smart generator with binary introspection
config := fuzzing.SmartGeneratorConfig{
    EnableIntrospection: true,
    CommonTestBinaries:  true,
    ValidateCommands:    true,
}
generator := fuzzing.NewSmartGeneratorWithEngine(seed, config)
script := generator.GenerateWithValidation()
```

Test binaries can support validation mode:
```go
if os.Getenv("MCP_SCRIPTTEST_VALIDATE_ONLY") == "1" {
    // Validate flags only, don't execute
    flag.Parse()
    if err := validateFlags(); err != nil {
        os.Exit(1) // Invalid
    }
    os.Exit(0) // Valid
}
```

See [SMART_GENERATOR.md](SMART_GENERATOR.md) for complete documentation.

## Coverage Data

The fuzzer uses Go's built-in coverage tools to:
1. Collect per-test coverage data
2. Identify code paths that increase coverage
3. Prioritize inputs that explore new code
4. Mutate successful inputs to find edge cases

Set `GOCOVERDIR` to enable coverage collection:

```bash
GOCOVERDIR=/tmp/coverage go test -fuzz=Fuzz
```

Analyze coverage data:

```bash
go tool covdata percent -i /tmp/coverage
```

## Advanced Usage

### Custom Options

```go
opts := &mcpscripttest.Options{
    Coverage: true,
    Trace:    true,
    Timeout:  30 * time.Second,
}

fuzzing.FuzzWithState(f, serverCmd, opts)
```

### Script Mutation Strategies

The fuzzer applies various mutation strategies:
- Adding/removing lines
- Modifying existing commands
- Duplicating successful patterns
- Swapping command order

### Coverage-Guided Optimization

The fuzzer maintains a corpus of high-scoring inputs based on:
- Coverage increase percentage
- Number of new packages covered
- Execution time (faster is better)

## Best Practices

1. **Build with coverage**: Ensure your binaries are built with `-cover` flag
2. **Set appropriate timeouts**: Prevent hanging tests from blocking fuzzing
3. **Monitor coverage trends**: Use verbose mode to track coverage improvements
4. **Combine with regular tests**: Use fuzzing to complement traditional tests
5. **Save interesting inputs**: Export high-coverage inputs as regular tests

## Examples

See the test files in this directory for complete examples of fuzzing different server configurations.