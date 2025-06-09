#!/bin/bash

# Run scripttest with isolated coverage collection

echo "=== Isolated Coverage Collection Demo ==="
echo

# Create a fresh coverage directory
COVERDIR=$(mktemp -d)
echo "Coverage directory: $COVERDIR"

# Run the test in a subshell to isolate environment
(
    # Set coverage directory only for this subshell
    export GOCOVERDIR="$COVERDIR"
    
    # Change to the test directory
    cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/mcpscripttest
    
    # Run the controlled coverage test
    go test -v -run TestControlledCoverage
)

echo
echo "=== Coverage Results ==="
echo "Coverage files created:"
find "$COVERDIR" -name "cov*" -type f | while read f; do
    echo "  $(basename $f)"
done

echo
echo "=== Coverage Analysis ==="
go tool covdata percent -i "$COVERDIR"

echo
echo "Coverage data saved in: $COVERDIR"
echo "To see detailed report:"
echo "  go tool covdata textfmt -i $COVERDIR -o coverage.txt"