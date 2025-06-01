# MCPScriptTest Fuzzing Summary

This fuzzing package provides coverage-guided fuzzing capabilities for MCPScriptTest, enabling automated test generation and coverage improvement.

## Implementation Status

✅ **Completed Features:**
- Coverage-guided fuzzing infrastructure
- Script generation with weighted command selection
- Mutation-based input generation
- Direct state-based fuzzing without file system overhead
- Integration with Go's built-in fuzzing system
- Coverage tracking and feedback loop
- Run() function for direct fuzzing

## Files

- `fuzzing.go` - Core fuzzing generator and scripttest integration
- `coverage_fuzzing.go` - Coverage feedback system and guided fuzzing
- `runner_fuzzing.go` - Direct execution with state management
- `README.md` - Usage documentation
- Test files demonstrating usage

## Usage

### Basic Fuzzing
```go
func FuzzMyServer(f *testing.F) {
    serverCmd := []string{"go", "run", "./server"}
    fuzzing.FuzzWithState(f, serverCmd, nil)
}
```

### Coverage-Guided Fuzzing
```bash
GOCOVERDIR=/tmp/coverage go test -fuzz=FuzzMyServer
```

### Direct Fuzzing
```go
err := fuzzing.Run(func(script string) error {
    return runScript(script)
}, fuzzing.DefaultRunOptions())
```

## Key Components

1. **FuzzGenerator** - Generates valid scripttest scripts
2. **CoverageFeedback** - Tracks coverage improvements
3. **CoverageGuidedFuzzer** - Combines generation and mutation
4. **RunWithState** - Executes scripts with state management

## Limitations

- MCPScripttestOptions doesn't have Coverage/Trace fields (these are handled differently)
- Some test examples need updating for the current API
- Coverage requires binaries built with `-cover` flag