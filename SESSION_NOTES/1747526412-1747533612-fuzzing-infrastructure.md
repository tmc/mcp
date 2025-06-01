# MCP Fuzzing Infrastructure Session Notes
**Date Range**: 2025-01-17 19:00:12 - 21:00:12 UTC (1747526412-1747533612)
**Topic**: Testing improvements and fuzzing infrastructure for MCP

## Overview
This session focused on significantly enhancing the MCP testing infrastructure with two main initiatives:
1. Improving test logging and helper organization
2. Building a comprehensive fuzzing infrastructure for mcpscripttest

## Test Logging Improvements

### Initial State
- Tests were outputting logs directly to stderr
- No distinction between normal and verbose test output
- Testing helpers were in the main package

### Changes Made
- **Created testutil package**: Moved all testing helpers from the main package to `/testutil/testing_helpers.go`
- **Implemented test-aware logging**: 
  - INFO level by default
  - DEBUG level with `-v` flag or `MCP_TEST_DEBUG=1` environment variable
  - Logs directed to `t.Log()` instead of stderr
- **Added structured logging support**: Using slog with custom test handler

### Key Files
- `/testutil/testing_helpers.go` - Core testing infrastructure
- `/testutil/go.mod` - Module definition
- Updated `/go.work` to include testutil module

## Fuzzing Infrastructure Development

### Requirements
The user requested three specific features for mcpscripttest:
1. Schema dumper tool to output valid scripttest command documentation
2. Fuzzing integration to generate test content using Go fuzzing system
3. Build patching/sandboxing to restrict stdlib functionality for security

### Implementation Timeline

#### Phase 1: Basic Fuzzing Framework
- Created `/exp/mcpscripttest/fuzzing/` subdirectory
- Implemented basic fuzzing generator in `fuzzing.go`
- Added FuzzScriptTest function for basic script generation
- Created schema dumper functionality

#### Phase 2: Coverage-Guided Fuzzing
- Implemented coverage feedback system in `coverage_fuzzing.go`
- Added CoverageGuidedFuzzer with feedback loops
- Created Run() function for direct state-based fuzzing
- Integrated with GOCOVERDIR for coverage collection

#### Phase 3: Visualization
- Built live visualization system in `visualization.go`
- Shows accepted/rejected scripts and statistics
- Terminal UI with color-coded output
- Real-time fuzzing progress display

#### Phase 4: Specialized Generators
- Created `specialized_generators.go` with configurable command weights
- Added MCPTraceGenerator for MCP-specific testing
- Implemented SafeFileOperationsGenerator for secure file operations
- Built constraint-based generation system

#### Phase 5: Smart Generator
- Developed `smart_generator.go` with binary introspection
- Added BinaryIntrospector for analyzing test binaries
- Created intelligent command generation based on binary capabilities
- Fixed hardcoded exec command issues

#### Phase 6: Test Binary Framework
- Implemented `test_binary_framework.go` with TestMainWithFuzzing
- Created cooperative fuzzing system where binaries participate
- Added multiple modes: Normal, Validate, Generate, Introspect
- Built example test_echo binary demonstrating the framework

### Key Files Created/Modified

#### Fuzzing Core
- `/exp/mcpscripttest/fuzzing/fuzzing.go` - Basic fuzzing generator
- `/exp/mcpscripttest/fuzzing/coverage_fuzzing.go` - Coverage-guided fuzzing
- `/exp/mcpscripttest/fuzzing/visualization.go` - Live fuzzing visualization
- `/exp/mcpscripttest/fuzzing/specialized_generators.go` - Specialized generators
- `/exp/mcpscripttest/fuzzing/smart_generator.go` - Binary-aware generator
- `/exp/mcpscripttest/fuzzing/binary_introspection.go` - Binary analysis
- `/exp/mcpscripttest/test_binary_framework.go` - Cooperative fuzzing framework

#### Examples and Tests
- `/exp/mcpscripttest/examples/test_echo/main.go` - Example cooperative binary
- Various test files demonstrating fuzzing capabilities

#### Module Structure
- Created `/exp/mcpscripttest/fuzzing/go.mod`
- Updated `/go.work` to include new modules

## Technical Challenges Solved

1. **Package Import Issues**: Fixed imports when moving to subdirectory
2. **Type References**: Updated MCPScripttestOptions to Options
3. **Environment Handling**: Changed from Env type to map[string]string
4. **Module Organization**: Created proper go.mod files and workspace structure
5. **String Parsing**: Fixed strconv.ParseFloat vs strings.ParseFloat issues
6. **Binary Introspection**: Implemented safe binary analysis for command generation

## TODO List

### Completed ✓
- [x] Move testing helpers to testutil package
- [x] Implement test-aware logging with verbose support
- [x] Create basic fuzzing generator
- [x] Add coverage-guided fuzzing
- [x] Build visualization system
- [x] Implement specialized generators
- [x] Create smart generator with binary introspection
- [x] Build test binary framework for cooperative fuzzing
- [x] Create example test binaries

### Pending
- [ ] Integrate TestMainWithFuzzing with smart generator
- [ ] Create comprehensive documentation for fuzzing system
- [ ] Add more example test binaries
- [ ] Build integration tests for the complete fuzzing pipeline
- [ ] Add sandbox/build patching for security constraints
- [ ] Create benchmarks comparing different generator strategies

## Recent File Changes
Since the testutil changes, significant modifications have been made to:
- Server infrastructure (`server.go`)
- Command-line tools (mcp-probe, mcp-shadow, mcpdiff, mcpspy)
- Test files across the codebase

## Architecture Notes

### Fuzzing System Architecture
```
mcpscripttest/
├── fuzzing/
│   ├── fuzzing.go              # Basic generator
│   ├── coverage_fuzzing.go     # Coverage feedback
│   ├── visualization.go        # Live UI
│   ├── specialized_generators.go # Domain-specific generation
│   ├── smart_generator.go      # Binary-aware generation
│   └── binary_introspection.go # Binary analysis
├── test_binary_framework.go    # Cooperative fuzzing
└── examples/
    └── test_echo/             # Example cooperative binary
```

### Key Design Decisions
1. **Modular generators**: Each generator serves a specific purpose
2. **Coverage feedback**: Guides fuzzing toward unexplored code paths
3. **Binary cooperation**: Test binaries can participate in fuzzing
4. **Live visualization**: Provides immediate feedback on fuzzing progress
5. **Flexible configuration**: Generators can be customized for different scenarios

## Next Steps
The logical next step would be to complete the integration between TestMainWithFuzzing and the smart generator to create a fully automated, intelligent fuzzing system that:
- Uses binary introspection to generate valid commands
- Validates commands using the binary's validate mode
- Provides coverage feedback to guide exploration
- Visualizes progress in real-time

This would create a powerful testing infrastructure that can automatically discover edge cases and improve test coverage across the MCP codebase.