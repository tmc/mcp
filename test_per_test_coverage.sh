#!/bin/bash
set -e

# Create coverage output directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
COVERAGE_DIR="coverage_${TIMESTAMP}"
mkdir -p "$COVERAGE_DIR"

echo "Running per-test coverage analysis..."
echo "Test,Coverage" > "$COVERAGE_DIR/summary.csv"

# Get all test names from the package
TEST_NAMES=$(go test -list . . | grep -E "^Test")

# Run each test individually
for test in $TEST_NAMES; do
    echo "Running: $test"
    
    # Run test with coverage
    if go test -run "^${test}$" -coverprofile="$COVERAGE_DIR/${test}.out" . 2>/dev/null; then
        # Get coverage percentage
        COVERAGE=$(go tool cover -func="$COVERAGE_DIR/${test}.out" | tail -1 | awk '{print $3}')
        echo "  Coverage: $COVERAGE"
        echo "$test,$COVERAGE" >> "$COVERAGE_DIR/summary.csv"
        
        # Generate HTML report for this test
        go tool cover -html="$COVERAGE_DIR/${test}.out" -o "$COVERAGE_DIR/${test}.html" 2>/dev/null || true
    else
        echo "  FAILED"
        echo "$test,FAILED" >> "$COVERAGE_DIR/summary.csv"
    fi
done

# Combine all coverage profiles
echo "mode: atomic" > "$COVERAGE_DIR/combined.out"
for profile in "$COVERAGE_DIR"/*.out; do
    if [[ "$profile" != "$COVERAGE_DIR/combined.out" ]]; then
        tail -n +2 "$profile" >> "$COVERAGE_DIR/combined.out" 2>/dev/null || true
    fi
done

# Generate combined HTML report
go tool cover -html="$COVERAGE_DIR/combined.out" -o "$COVERAGE_DIR/combined.html" 2>/dev/null || true

# Show summary
echo -e "\n=== Coverage Summary ==="
column -t -s, "$COVERAGE_DIR/summary.csv"
echo -e "\nReports saved in: $COVERAGE_DIR"
echo "Combined coverage: $COVERAGE_DIR/combined.html"