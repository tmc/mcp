#!/bin/bash

# Complete example showing how to run scripttest with coverage
# and prevent GOCOVERDIR from propagating to subdirectories

echo "=== Complete Coverage Example ==="
echo

# Create an isolated coverage directory
MAINDIR=$(mktemp -d)
export GOCOVERDIR="$MAINDIR"

echo "Main coverage directory: $GOCOVERDIR"
echo

# Run tests in current directory to avoid propagation
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/mcpscripttest

# Run different coverage tests
echo "1. Running explicit coverage test..."
go test -v -run TestExplicitCoverage

echo
echo "2. Running controlled coverage test..."
go test -v -run TestControlledCoverage

echo
echo "=== Coverage Analysis ==="

# Find all coverage files
echo "Coverage files created:"
find "$MAINDIR" -name "cov*" -type f | while read f; do
    echo "  $f"
done

echo
echo "Coverage summary:"
go tool covdata percent -i "$MAINDIR"

echo
echo "To generate detailed report:"
echo "  go tool covdata textfmt -i $MAINDIR -o coverage.txt"
echo "  cat coverage.txt"

echo
echo "All coverage data saved in: $MAINDIR"