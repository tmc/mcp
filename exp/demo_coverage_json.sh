#!/bin/bash
# Demo: Generate a Codecov JSON file for a single test

echo "Generating Codecov JSON coverage for TestClientNotificationHandling..."

# Run the test with coverage
cd /Volumes/tmc/go/src/github.com/tmc/mcp
go test -coverprofile=demo_coverage.out -run TestClientNotificationHandling

# Convert to Codecov JSON using the demo converter
go run exp/demo_codecov_converter.go demo_coverage.out demo_codecov.json

echo "Coverage data written to demo_codecov.json"
echo "Preview:"
head -50 demo_codecov.json