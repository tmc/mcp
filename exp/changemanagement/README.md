# Intelligent Change Management System

This is an implementation of an AI-powered change management system that automates the entire development workflow from change description to resolution.

## Overview

The system takes a natural language description of a change and:
1. Analyzes the change to understand its type, risk, and impact
2. Finds tests that will be affected
3. Generates comprehensive documentation
4. Creates test mutations to improve coverage
5. Orchestrates the entire workflow

## Components

### Core Tools

1. **mcp-change-analyze** - Analyzes natural language change descriptions
2. **mcp-test-find** - Finds tests affected by changes
3. **mcp-doc-gen** - Generates documentation for changes
4. **mcp-test-mutate** - Creates test variations
5. **mcp-change-execute** - Orchestrates the entire workflow

### Libraries

- `analyzer.go` - Change analysis engine
- `testfinder.go` - Test discovery logic
- `docgen.go` - Documentation generator
- `mutator.go` - Test mutation engine

## Installation

```bash
# Install all tools
cd exp/changemanagement
go install ./cmd/...
```

## Quick Start

### 1. Analyze a Change

```bash
mcp-change-analyze -description "Add OAuth2 authentication to all API endpoints"
```

### 2. Find Affected Tests

```bash
mcp-change-analyze -description "Add OAuth2 authentication" -output analysis.json
mcp-test-find -change analysis.json -codebase .
```

### 3. Generate Documentation

```bash
mcp-doc-gen -change analysis.json -output docs/
```

### 4. Create Test Mutations

```bash
mcp-test-mutate -test example_test.go -output mutations/
```

### 5. Execute Complete Workflow

```bash
mcp-change-execute -description "Add OAuth2 authentication to all API endpoints" \
  -codebase . \
  -output change-output/
```

## Example Workflow

Here's a complete example of adding OAuth2 authentication:

```bash
# 1. Describe the change
CHANGE="Add OAuth2 authentication support to all API endpoints with token refresh"

# 2. Execute the change workflow
mcp-change-execute -description "$CHANGE" -codebase ~/myproject

# 3. Review the output
cd change-output/
ls -la

# Output includes:
# - analysis.json          # Change analysis results
# - affected_tests.json    # List of affected tests
# - docs/                  # Generated documentation
# - mutations/             # Test mutations
# - change_report.json     # Complete execution report
```

## Change Types

The system recognizes several types of changes:
- **feature** - New functionality
- **refactoring** - Code reorganization
- **bugfix** - Bug fixes
- **performance** - Performance improvements
- **security** - Security enhancements
- **migration** - Data or system migrations

## Risk Levels

Changes are classified by risk:
- **low** - Minimal impact, safe to deploy
- **medium** - Moderate impact, needs testing
- **high** - Significant impact, requires careful rollout

## Documentation Generation

The system generates several types of documentation:
- Overview document
- Migration guide (for breaking changes)
- API documentation (for API changes)
- Security notes (for security changes)
- Release notes

## Test Mutation Strategies

The mutator supports several strategies:
- **reorder** - Reorders test commands
- **fuzz** - Fuzzes input values
- **timing** - Modifies timing and delays
- **error** - Injects error conditions

## Advanced Usage

### Custom Output Formats

```bash
# Generate text format analysis
mcp-change-analyze -description "..." -format text

# Generate HTML documentation
mcp-doc-gen -change analysis.json -format html
```

### Dry Run Mode

```bash
# Analyze without executing changes
mcp-change-execute -description "..." -dry-run
```

### Verbose Output

```bash
# See detailed execution logs
mcp-change-execute -description "..." -verbose
```

## Architecture

The system follows a modular architecture:
1. **Analysis Layer** - Natural language processing and pattern matching
2. **Discovery Layer** - Code and test discovery
3. **Generation Layer** - Documentation and test generation
4. **Orchestration Layer** - Workflow coordination

## Contributing

To add new features:
1. Add new change types in `analyzer.go`
2. Extend pattern matching for better analysis
3. Add new documentation templates in `docgen.go`
4. Create new mutation strategies in `mutator.go`

## Future Enhancements

Planned improvements include:
- AI-powered code generation
- Automatic test updates
- Integration with CI/CD systems
- Real-time monitoring and feedback
- Cross-language support

## License

This is part of the MCP project and follows the same license terms.