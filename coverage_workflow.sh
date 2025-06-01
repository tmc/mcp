#!/bin/bash
# Coverage Analysis Workflow Demo
# This script demonstrates the complete coverage analysis workflow

set -e

echo "=== MCP Coverage Analysis Workflow ==="
echo

# Create output directories
mkdir -p coverage_output/{individual,combined,reports}

# Step 1: Run a subset of tests individually
echo "Step 1: Analyzing individual test coverage..."
./exp/covtest/covtest -pkg . -out coverage_output/analysis \
    -codecov coverage_output/individual -per-test \
    -run "TestClient.*|TestServer.*" || true

echo
echo "Step 2: Converting test coverage to Codecov JSON..."
# Run a test with traditional coverage
go test -coverprofile=coverage_output/coverage.out -run TestClientNotificationHandling

# Convert using our demo converter
go run demo_codecov_converter.go coverage_output/coverage.out \
    coverage_output/reports/client_notification.json

echo
echo "Step 3: Generating combined coverage report..."
# Generate binary coverage for all tests
mkdir -p coverage_output/binary
GOCOVERDIR=coverage_output/binary go test || true

# Convert to combined Codecov JSON
./exp/cov2codecov/cov2codecov -input coverage_output/binary \
    -output coverage_output/combined/all_tests.json -json || true

echo
echo "Step 4: Comparing coverage (demo)..."
# Create two different coverage datasets
mkdir -p coverage_output/{before,after}
GOCOVERDIR=coverage_output/before go test -run "TestClient.*" || true
GOCOVERDIR=coverage_output/after go test -run "Test.*" || true

# Compare them
./exp/covdiff/covdiff -base coverage_output/before -new coverage_output/after || true

echo
echo "=== Coverage Analysis Complete ==="
echo
echo "Results available in:"
echo "  - Individual test coverage: coverage_output/individual/"
echo "  - Coverage reports: coverage_output/reports/"
echo "  - Combined coverage: coverage_output/combined/"
echo
echo "Example Codecov JSON output:"
head -30 coverage_output/reports/client_notification.json || true

echo
echo "To view test contributions:"
echo "  cat coverage_output/analysis/test-contributions.json | jq '.'"