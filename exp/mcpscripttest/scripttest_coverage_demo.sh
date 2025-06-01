#!/bin/bash

# Demo script showing scripttest's coverage collection across multiple binaries

echo "=== MCP Scripttest Coverage Demo ==="
echo "This demo shows how scripttest collects coverage from multiple tools"
echo

# Create a temporary directory for our demo
DEMO_DIR=$(mktemp -d -t mcp-scripttest-coverage-demo)
echo "Demo directory: $DEMO_DIR"
echo

# Set up coverage directory
export GOCOVERDIR="$DEMO_DIR/coverage"
mkdir -p "$GOCOVERDIR"
echo "Coverage directory: $GOCOVERDIR"
echo

# Run the coverage tests
echo "Running scripttest tests that use multiple MCP tools..."
go test -v -run "TestScripttestCoverage" ./...

echo
echo "=== Coverage Analysis ==="

# Check coverage data
cd "$DEMO_DIR"
if [ -d coverage ]; then
    echo "Coverage files found:"
    ls -la coverage
    echo
    
    # Analyze coverage by package
    echo "Coverage by package:"
    go tool covdata percent -i coverage
    echo
    
    # Generate text report
    echo "Generating detailed coverage report..."
    go tool covdata textfmt -i coverage -o coverage.txt
    
    # Show coverage summary
    echo "Coverage summary:"
    grep -E "^github.com/tmc/mcp" coverage.txt | head -20
fi

# Cleanup
echo
echo "Demo complete!"
echo "Coverage data saved in: $DEMO_DIR"
echo "To view full coverage report: cat $DEMO_DIR/coverage.txt"