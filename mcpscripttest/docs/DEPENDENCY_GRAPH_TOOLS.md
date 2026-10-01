# Dependency Graph Tools

This document describes the suite of tools for analyzing test-to-program dependencies and coverage in mcpscripttest.

## Overview

The dependency graph tools help you:
- Understand which tests cover which programs
- Find programs with no test coverage
- Analyze test coverage by program
- Generate various graph formats for visualization
- Perform advanced graph queries using the golang digraph tool

## Tools

### testgraph

Combines testcallgraph analysis with digraph output format for Unix-style composability.

```bash
# Generate digraph format (default)
testgraph testdata/ > graph.digraph

# Generate JSON format for further processing
testgraph -format json testdata/ > graph.json

# Generate DOT format for visualization
testgraph -format dot testdata/ > graph.dot
```

### depgraph

Transforms callgraph data into various dependency graph formats.

```bash
# Basic dependency graph
depgraph -input graph.json

# Reverse dependencies (program-to-test)
depgraph -input graph.json -direction program-to-test

# Transitive closure
depgraph -input graph.json -type transitive

# Adjacency matrix
depgraph -input graph.json -format matrix

# Grouped visualization
depgraph -input graph.json -format dot -group
```

### coverage-by-program

Analyzes test coverage per program based on dependency graph.

```bash
# Basic coverage analysis
coverage-by-program -depgraph deps.json -coverage /tmp/coverage

# Show uncovered programs
coverage-by-program -depgraph deps.json -coverage /tmp/coverage -show-uncovered

# Generate markdown report
coverage-by-program -depgraph deps.json -coverage /tmp/coverage -format markdown

# Include detailed statistics
coverage-by-program -depgraph deps.json -coverage /tmp/coverage -stats -verbose
```

## Workflow Examples

### 1. Find Programs Without Tests

```bash
# Generate callgraph
testgraph -format json testdata/ > callgraph.json

# Create dependency graph
depgraph -input callgraph.json > deps.digraph

# Find all programs
depgraph -input callgraph.json | digraph sinks > all-programs.txt

# Find tested programs
depgraph -input callgraph.json | digraph sinks | \
  while read prog; do
    if depgraph -input callgraph.json -direction program-to-test | \
       digraph successors "$prog" | grep -q .; then
      echo "$prog"
    fi
  done > tested-programs.txt

# Find untested programs
comm -23 <(sort all-programs.txt) <(sort tested-programs.txt)
```

### 2. Analyze Test Coverage

```bash
# Generate dependency graph
testgraph -format json testdata/ > callgraph.json
depgraph -input callgraph.json -format json > deps.json

# Analyze coverage
coverage-by-program -depgraph deps.json -coverage /tmp/coverage -format markdown > report.md
```

### 3. Find Test Dependencies

```bash
# Which programs does test1.txt use?
testgraph testdata/ | digraph successors test1.txt

# Which tests cover mcpdiff?
testgraph testdata/ | digraph predecessors mcpdiff

# Find all paths from test1.txt to mcpdiff
testgraph testdata/ | digraph allpaths test1.txt mcpdiff
```

### 4. Visualize Dependencies

```bash
# Generate DOT file
depgraph -input callgraph.json -format dot -group > deps.dot

# Create PNG visualization
dot -Tpng deps.dot -o deps.png

# Create SVG for web
dot -Tsvg deps.dot -o deps.svg
```

## Advanced Usage

### Integration with Coverage Hotspots

Combine with coverage-hotspots to find untested code in critical programs:

```bash
# Find programs with low coverage
coverage-by-program -depgraph deps.json -coverage /tmp/coverage -format json | \
  jq '.programs[] | select(.CoveragePercent < 50)'

# Find hotspots in those programs
coverage-hotspots -coverage /tmp/coverage -threshold 100 | \
  grep -f <(low-coverage-programs)
```

### Custom Queries with digraph

The digraph tool supports various graph operations:

```bash
# Find cycles in dependencies
testgraph testdata/ | digraph cycles

# Find strongly connected components
testgraph testdata/ | digraph scc

# Find shortest path between nodes
testgraph testdata/ | digraph somepath test1.txt mcpdiff

# Find all nodes reachable from a test
testgraph testdata/ | digraph forward test1.txt
```

### Pipeline Integration

Use in CI/CD pipelines to track coverage trends:

```bash
#!/bin/bash
# coverage-check.sh

# Generate current coverage report
testgraph -format json testdata/ > callgraph.json
depgraph -input callgraph.json -format json > deps.json
coverage-by-program -depgraph deps.json -coverage coverage/ -format json > current.json

# Compare with baseline
if [ -f baseline.json ]; then
  jq -r '.programs[] | "\(.Program): \(.CoveragePercent)%"' current.json > current.txt
  jq -r '.programs[] | "\(.Program): \(.CoveragePercent)%"' baseline.json > baseline.txt
  
  # Show differences
  diff baseline.txt current.txt || true
fi

# Update baseline
cp current.json baseline.json
```

## Best Practices

1. **Regular Analysis**: Run dependency analysis regularly to catch coverage gaps early
2. **Baseline Tracking**: Keep baseline coverage data to track trends
3. **Visualization**: Use DOT visualizations for code reviews and documentation
4. **Automation**: Integrate into CI/CD pipelines for continuous monitoring
5. **Focused Testing**: Use dependency information to run only relevant tests when code changes

## Troubleshooting

### No Coverage Data

If coverage-by-program shows 0% for all programs:
1. Ensure coverage data was collected with `-cover` flag
2. Check that GOCOVERDIR is set correctly
3. Verify package names match between tests and programs

### Missing Dependencies

If expected dependencies don't appear:
1. Check that test files use `exec` commands
2. Verify program names are correctly extracted
3. Look for typos in command names

### Performance Issues

For large test suites:
1. Use `-verbose` to identify slow operations
2. Consider filtering test files before analysis
3. Cache intermediate results (callgraph.json, deps.json)