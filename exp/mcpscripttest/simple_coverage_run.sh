#!/bin/bash

# The simplest way to run a scripttest and show coverage

# 1. Create a coverage directory
COVERDIR=$(mktemp -d)
echo "Coverage directory: $COVERDIR"

# 2. Run the test with GOCOVERDIR set
GOCOVERDIR=$COVERDIR go test -v -run TestScripttestCoverageAcrossBinaries

# 3. Show coverage results
echo
echo "=== Coverage Results ==="
go tool covdata percent -i $COVERDIR

# 4. Generate detailed report
go tool covdata textfmt -i $COVERDIR -o $COVERDIR/coverage.txt
echo
echo "Detailed coverage saved to: $COVERDIR/coverage.txt"
echo "First few lines:"
head -n 10 $COVERDIR/coverage.txt

# Keep the directory for inspection
echo
echo "Coverage data kept in: $COVERDIR"