# Coverage Comparison: mcpscripttest vs Other Tests

## Summary

This document compares test coverage between mcpscripttest-based tests and other test types in the MCP Go implementation.

## mcpscripttest Coverage
- **Package**: `github.com/tmc/mcp/exp/mcpscripttest`
- **Coverage**: 54.5% of statements
- **Test Type**: Script-based integration tests using txtar format
- **Tests Run**: 2 main test functions covering tool integration and coverage collection
- **Notable**: Tests successfully integrate coverage from multiple binaries (mcpdiff, mcpspy, mcpcat)

## Other Package Coverage

### Core MCP Package
- **Package**: `github.com/tmc/mcp` (main)
- **Coverage**: 53.0% of statements  
- **Test Type**: Unit and integration tests

### Protocol Implementation
- **Package**: `github.com/tmc/mcp/modelcontextprotocol`
- **Coverage**: ~65% of statements (main package)
- **Package**: `github.com/tmc/mcp/modelcontextprotocol/draft`
- **Coverage**: 89.4% of statements
- **Test Type**: Protocol validation, marshaling/unmarshaling tests

### Command-Line Tools
- **mcpcat**: 62.2% of statements
- **mcp-shadow**: 18.8% of statements
- **mcp-replay**: 10.7% of statements
- **Most cmd packages**: 0.0% of statements (main functions, harder to test)

### Internal Utilities
- **jsonrpc2shim**: 100.0% of statements
- **jsonrpc2util**: 20.1% of statements
- **testing**: 63.9% of statements
- **testutil**: 52.7% of statements

## Key Findings

1. **Comparable Core Coverage**: mcpscripttest (54.5%) vs main package (53.0%) have very similar coverage levels
2. **Integration vs Unit Testing**: mcpscripttest focuses on tool integration while other tests focus on unit-level testing
3. **Protocol Coverage**: The draft protocol package has excellent coverage (89.4%), showing thorough testing of new features
4. **Command Coverage**: Most command-line tools have low coverage, which is expected for main functions
5. **Cross-Binary Coverage**: mcpscripttest successfully demonstrates coverage collection across multiple tool binaries

## Coverage Collection Success

The mcpscripttest framework successfully:
- Builds tools with coverage instrumentation (`go install -cover`)
- Collects coverage data from multiple binaries during script execution
- Aggregates coverage data from 3 different tools (mcpdiff, mcpspy, mcpcat)
- Provides 54.5% coverage through integration testing

## Conclusion

The mcpscripttest framework provides valuable integration testing coverage (54.5%) that complements the unit test coverage in other packages (53.0% average). The framework successfully demonstrates:
- Tool installation and PATH management
- Cross-binary coverage collection
- Script-based testing of real tool interactions
- Integration testing of the MCP protocol through actual tool usage

Both testing approaches are valuable: unit tests provide detailed coverage of individual functions, while mcpscripttest provides end-to-end integration coverage of tool workflows.