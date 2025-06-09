#!/bin/bash

# Demo script to show coverage-enabled tool building

# Create a temporary directory for our demo
DEMO_DIR=$(mktemp -d -t mcp-coverage-demo)
echo "Demo directory: $DEMO_DIR"

# Set up coverage directory
export GOCOVERDIR="$DEMO_DIR/coverage"
mkdir -p "$GOCOVERDIR"

# Run our integration test
echo "Running tool coverage integration test..."
go test -v -run TestToolsCoverageIntegration

# Check coverage data
echo "Checking coverage data..."
cd "$DEMO_DIR"
if [ -d coverage ]; then
    echo "Coverage files found:"
    ls coverage
    
    # Analyze coverage
    echo "Coverage analysis:"
    go tool covdata percent -i coverage
fi

# Cleanup
cd -
rm -rf "$DEMO_DIR"