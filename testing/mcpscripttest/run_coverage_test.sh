#!/bin/bash

# Complete example of running scripttest with coverage

# Setup
COVERDIR=$(mktemp -d)
export GOCOVERDIR=$COVERDIR

echo "=== Running MCP Scripttest with Coverage ==="
echo "Coverage directory: $COVERDIR"
echo

# Run test
echo "Running test..."
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/mcpscripttest
go test -v -run TestScripttestCoverageAcrossBinaries

# Find coverage files
echo
echo "=== Coverage Files ==="
find $COVERDIR -type f -name "cov*" | while read f; do
    echo "  $(basename $f)"
done

# Try to analyze coverage
echo
echo "=== Coverage Analysis ==="
# The test creates its own coverage subdirectory, find it
ACTUAL_COVERDIR=$(find $COVERDIR -type d -name "coverage" | head -1)
if [ -n "$ACTUAL_COVERDIR" ]; then
    echo "Found coverage data in: $ACTUAL_COVERDIR"
    go tool covdata percent -i "$ACTUAL_COVERDIR"
else
    echo "Trying base directory: $COVERDIR"
    go tool covdata percent -i "$COVERDIR"
fi

echo
echo "Coverage data location: $COVERDIR"
echo "To explore further:"
echo "  find $COVERDIR -type f -name 'cov*'"