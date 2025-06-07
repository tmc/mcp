# Test Coverage Visualization Plan

## Overview

This plan outlines how to enhance the MCP project's test coverage analysis by integrating testcallgraph capabilities with visualization similar to Codecov's approach. The goal is to provide developers with deeper insights into the relationship between tests and source code.

## Current State Analysis

### Existing Components

1. **Coverage Infrastructure** (`exp/mcpscripttest/coverage.go`)
   - Per-test coverage collection
   - GOCOVERDIR environment management
   - Coverage data merging capabilities

2. **Call Graph Analysis** (`exp/mcpscripttest/testcallgraph/prototype.go`)
   - Static call graph from SSA analysis
   - Dynamic execution trace capabilities
   - Test-to-source mapping
   - Proximity analysis for finding tests near uncovered code

3. **Visualization Concepts**
   - Graphviz DOT generation for call graphs
   - Test impact analysis
   - Coverage gap detection

## Codecov-Inspired Features to Implement

### 1. Source Code Annotation

Like Codecov, we want to annotate source code with:
- Coverage percentage per file
- Line-by-line coverage indicators
- Test mapping showing which tests cover each line
- Visual highlighting (green for covered, red for uncovered, yellow for partial)

### 2. Coverage Trends

Track coverage changes over time:
- Per-commit coverage delta
- Historical trends graph
- Coverage regression alerts
- Pull request coverage impact

### 3. Interactive Web Interface

Create a web-based visualization similar to Codecov:

```
┌─ mcp/cmd/mcpdiff/main.go ──────────────────────────┐
│ Coverage: 73.4%                                     │
├─────────────────────────────────────────────────────┤
│ 15  ✓ func main() {              // 3 tests        │
│ 16  ✓     var (                                    │
│ 17  ✓         ignoreTimestamps = flag.Bool(...)   │
│ 18  ✗         ignoreIDs = flag.Bool(...)          │
│ 19  ⚡         verbose = flag.Bool(...)            │
│ 20  ✓     )                                        │
│                                                    │
│ Legend: ✓ Covered  ✗ Uncovered  ⚡ Partial         │
└─────────────────────────────────────────────────────┘
```

### 4. Test Impact Analysis

Show the impact of each test:
- Which functions are called
- Coverage percentage contribution
- Execution time
- Redundancy detection

## Implementation Plan

### Phase 1: Data Collection Enhancement (Week 1-2)

1. **Extend Coverage Collection**
   ```go
   type EnhancedCoverage struct {
       LineHits     map[string]map[int]int
       TestMapping  map[string][]TestInfo
       CallGraphs   map[string]*callgraph.Graph
   }
   ```

2. **Integrate with testcallgraph**
   - Combine static and dynamic analysis
   - Create test-to-line mapping
   - Build execution traces per test

3. **Create Coverage Database**
   - SQLite or similar for persistence
   - Track historical data
   - Support incremental updates

### Phase 2: Analysis Engine (Week 3-4)

1. **Coverage Calculator**
   - File-level coverage percentages
   - Function-level coverage
   - Package-level aggregation
   - Change impact analysis

2. **Test Impact Analyzer**
   - Calculate unique coverage per test
   - Identify redundant tests
   - Suggest optimal test execution order
   - Find coverage gaps

3. **Proximity Analyzer Enhancement**
   - Find tests closest to uncovered code
   - Suggest test modifications
   - Generate test templates

### Phase 3: Visualization Layer (Week 5-6)

1. **Web Server**
   ```go
   type CoverageServer struct {
       coverage *EnhancedCoverage
       router   *http.ServeMux
   }
   ```

2. **REST API**
   - `/api/coverage/{file}` - Get file coverage
   - `/api/tests/{test}` - Get test impact
   - `/api/trends` - Get historical data
   - `/api/gaps` - Get coverage gaps

3. **Frontend Components**
   - React or Vue.js application
   - Source code viewer with annotations
   - Interactive call graphs
   - Coverage trends charts
   - Test impact visualizations

### Phase 4: Integration (Week 7-8)

1. **CI/CD Integration**
   - GitHub Actions workflow
   - Coverage upload mechanism
   - PR comment integration
   - Status checks

2. **Command-Line Tools**
   ```bash
   # Generate coverage report
   mcp-coverage report --format=html
   
   # Find tests for uncovered code
   mcp-coverage suggest --file=main.go --line=45
   
   # Analyze test impact
   mcp-coverage impact --test="TestBasicDiff"
   ```

3. **IDE Plugins**
   - VS Code extension
   - Coverage indicators in gutter
   - Test navigation
   - Quick fix suggestions

## Technical Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│   Test Runner   │────▶│ Coverage Engine  │────▶│  Visualization  │
│                 │     │                  │     │     Server      │
│ - scripttest    │     │ - Collection     │     │                 │
│ - Go test       │     │ - Analysis       │     │ - Web UI        │
│ - Coverage      │     │ - CallGraph      │     │ - API           │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │   Data Store     │
                        │                  │
                        │ - Coverage DB    │
                        │ - Historical     │
                        └──────────────────┘
```

## Example Usage

### 1. Running Tests with Enhanced Coverage

```bash
# Run tests with call graph analysis
COVERAGE_MODE=enhanced go test ./...

# Run specific scripttest with visualization
mcpscripttest -cover -visualize coverage_test.txt
```

### 2. Viewing Results

```bash
# Start web server
mcp-coverage serve --port=8080

# Open browser to view results
open http://localhost:8080
```

### 3. CI Integration

```yaml
name: Coverage Analysis
on: [push, pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
      
      - name: Run tests with coverage
        run: |
          go test -coverprofile=coverage.out ./...
          mcpscripttest -cover ./testdata/**/*.txt
          
      - name: Upload coverage
        run: mcp-coverage upload --branch=${{ github.ref }}
      
      - name: Comment on PR
        if: github.event_name == 'pull_request'
        run: mcp-coverage comment --pr=${{ github.event.number }}
```

## Benefits

1. **Better Test Understanding**
   - Clear visualization of test impact
   - Easy identification of redundant tests
   - Optimal test execution strategies

2. **Improved Code Quality**
   - Visual coverage gaps
   - Test suggestions for uncovered code
   - Historical trend tracking

3. **Developer Productivity**
   - Quick navigation between tests and code
   - IDE integration for immediate feedback
   - Automated test generation suggestions

4. **Team Collaboration**
   - Shared coverage reports
   - PR coverage requirements
   - Coverage goals and tracking

## Next Steps

1. Create proof-of-concept for Phase 1
2. Gather feedback from team
3. Iterate on design based on feedback
4. Begin implementation
5. Release beta version for testing
6. Full rollout with documentation

## Conclusion

By combining the existing coverage and testcallgraph capabilities with Codecov-inspired visualizations, we can create a powerful tool that helps developers write better tests, understand code coverage deeply, and maintain high-quality codebases.