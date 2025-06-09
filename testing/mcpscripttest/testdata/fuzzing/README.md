# Fuzzing Tests for mcpscripttest

This directory contains mcpscripttest tests that verify the fuzzing functionality works correctly.

## Test Files

### basic_fuzzing_test.txt
Tests basic fuzzing functionality including:
- FuzzScriptTest function availability
- Script generation capabilities
- Valid scripttest format generation

### coverage_guided_fuzzing_test.txt
Tests coverage-guided fuzzing:
- Coverage feedback initialization
- New coverage discovery
- Coverage file creation
- Run function with coverage

### specialized_generators_test.txt
Tests specialized script generators:
- MCP-focused generator (mcp-trace, mcp-send, etc.)
- Safe file operations generator
- Custom generator configuration
- Generator validation

### smart_generator_test.txt
Tests smart generator with binary introspection:
- Binary analysis capabilities
- Command generation based on introspection
- Validation and regeneration

### cooperative_fuzzing_test.txt
Tests cooperative fuzzing with test binaries:
- Binary introspection mode
- Binary generation mode
- Binary validation mode
- Full integration with smart generator

### visualization_test.txt
Tests live fuzzing visualization:
- Accepted/rejected script display
- Statistics tracking
- Coverage visualization

### example_usage_test.txt
Provides examples of using fuzzing within mcpscripttest:
- Simple fuzzing tests
- Coverage-enabled fuzzing
- Standalone fuzzing with Run()
- Specialized generators
- Cooperative fuzzing

## Test Servers

The `test_servers/` directory contains simple MCP servers used for testing.

## Fuzzing Tools

The `fuzzing_tools/` directory contains utilities for generating and testing fuzzing capabilities.

## Running the Tests

To run all fuzzing tests:

```bash
cd exp/mcpscripttest
go test -v -run TestFuzzingScripts
```

To run a specific test:

```bash
go test -v -run TestFuzzingScripts/basic_fuzzing
```

## Integration with Go Fuzzing

These tests verify that the fuzzing system integrates properly with Go's built-in fuzzing framework. The tests can be run as part of regular test suites or with fuzzing enabled:

```bash
# Regular test
go test -v ./fuzzing

# With fuzzing
go test -fuzz=Fuzz -fuzztime=30s ./fuzzing
```

## Coverage Analysis

Enable coverage collection during fuzzing:

```bash
export GOCOVERDIR=/tmp/coverage
go test -v ./fuzzing
go tool covdata percent -i /tmp/coverage
```