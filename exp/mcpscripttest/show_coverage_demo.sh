#!/bin/bash

# Simple demo to show coverage collection from a scripttest

echo "=== Coverage Demo for Scripttest ==="
echo

# Create temporary directory for coverage
DEMO_DIR=$(mktemp -d)
export GOCOVERDIR="$DEMO_DIR/coverage"
mkdir -p "$GOCOVERDIR"

echo "Coverage directory: $GOCOVERDIR"
echo

# Run just the coverage example test
echo "Running scripttest with coverage-enabled tools..."
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/mcpscripttest
go test -v -run TestScripttestCoverageAcrossBinaries

echo
echo "=== Coverage Files Created ==="
ls -la "$GOCOVERDIR"

echo
echo "=== Coverage Analysis ==="
go tool covdata percent -i "$GOCOVERDIR"

echo
echo "To see detailed coverage:"
echo "  go tool covdata textfmt -i $GOCOVERDIR -o coverage.txt"
echo "  cat coverage.txt"

# Cleanup
rm -rf "$DEMO_DIR"