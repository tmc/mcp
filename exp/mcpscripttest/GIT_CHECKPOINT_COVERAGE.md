# Git-Checkpoint Coverage Implementation

This document shows how to apply the improved bash coverage techniques to git-checkpoint.

## 1. Non-Invasive Coverage Wrapper

Create a coverage wrapper that doesn't modify the original script:

```bash
#!/bin/bash
# git-checkpoint-coverage.sh - Coverage wrapper for git-checkpoint

# Configuration
COVERAGE_DIR="${COVERAGE_DIR:-$HOME/.git-checkpoint/coverage}"
COVERAGE_ENABLED="${COVERAGE_ENABLED:-true}"
SCRIPT_PATH="$(dirname "$0")/git-checkpoint.sh"

# Ensure coverage directory exists
mkdir -p "$COVERAGE_DIR"

# Create trace file with timestamp
TRACE_FILE="$COVERAGE_DIR/trace-$(date +%Y%m%d-%H%M%S)-$$.txt"

# Function to process trace file after execution
process_trace() {
    local trace_file=$1
    local coverage_file="$COVERAGE_DIR/coverage-latest.json"
    
    # Parse trace and generate coverage data
    {
        echo "{"
        echo "  \"timestamp\": \"$(date -Iseconds)\","
        echo "  \"script\": \"git-checkpoint.sh\","
        echo "  \"executed_lines\": ["
        
        # Extract unique line numbers
        grep -o '^+[^:]*:[0-9]*:' "$trace_file" | 
            sed 's/+.*:\([0-9]*\):.*/\1/' | 
            sort -n | uniq | 
            sed '$ ! s/$/,/'
        
        echo "  ],"
        echo "  \"functions\": ["
        
        # Extract function calls
        grep -o '^+[^:]*:[0-9]*:[^:()]*():' "$trace_file" |
            sed 's/.*:\([^:()]*\)():.*/"\1"/' |
            sort | uniq |
            sed '$ ! s/$/,/'
        
        echo "  ]"
        echo "}"
    } > "$coverage_file"
}

# Main execution
if [[ "$COVERAGE_ENABLED" == "true" ]]; then
    # Open trace file descriptor
    exec 3>"$TRACE_FILE"
    
    # Set up enhanced bash tracing
    export BASH_XTRACEFD=3
    export PS4='+${BASH_SOURCE[0]##*/}:${LINENO}:${FUNCNAME[0]:+${FUNCNAME[0]}():} '
    
    # Enable tracing
    set -x
    
    # Source the script with tracing enabled
    source "$SCRIPT_PATH"
    
    # Execute the main function or command
    git_checkpoint "$@"
    EXIT_CODE=$?
    
    # Disable tracing
    set +x
    exec 3>&-
    
    # Process the trace file
    process_trace "$TRACE_FILE"
    
    # Optional: compress old traces
    find "$COVERAGE_DIR" -name "trace-*.txt" -mtime +7 -exec gzip {} \;
    
    exit $EXIT_CODE
else
    # Run without coverage
    source "$SCRIPT_PATH"
    git_checkpoint "$@"
fi
```

## 2. Safe Test Runner with Coverage

Create a test runner that safely collects coverage:

```bash
#!/bin/bash
# test-with-coverage.sh - Run tests with coverage collection

set -euo pipefail

# Configuration
COVERAGE_DIR="$(pwd)/.coverage"
TEST_DIR="$(pwd)/tests"
REPORT_FILE="$COVERAGE_DIR/report.html"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

# Initialize coverage directory
init_coverage() {
    rm -rf "$COVERAGE_DIR"
    mkdir -p "$COVERAGE_DIR"
    echo "Coverage directory initialized: $COVERAGE_DIR"
}

# Run a single test with coverage
run_test_with_coverage() {
    local test_file=$1
    local test_name=$(basename "$test_file" .bats)
    local trace_file="$COVERAGE_DIR/trace-$test_name.txt"
    
    echo -e "\n${GREEN}Running test: $test_name${NC}"
    
    # Set up coverage environment
    export COVERAGE_ENABLED=true
    export COVERAGE_DIR="$COVERAGE_DIR"
    
    # Run test with timeout and resource limits
    (
        ulimit -t 60     # 60 second CPU time limit
        ulimit -m 100000 # 100MB memory limit
        
        exec 3>"$trace_file"
        export BASH_XTRACEFD=3
        export PS4='+${BASH_SOURCE[0]##*/}:${LINENO}:${FUNCNAME[0]:+${FUNCNAME[0]}():} '
        
        timeout --kill-after=5s 30s bats "$test_file"
    )
    
    local test_result=$?
    
    if [[ $test_result -eq 0 ]]; then
        echo -e "${GREEN}✓ Test passed${NC}"
    else
        echo -e "${RED}✗ Test failed (exit code: $test_result)${NC}"
    fi
    
    return $test_result
}

# Generate coverage report
generate_report() {
    echo -e "\n${GREEN}Generating coverage report...${NC}"
    
    local total_lines=$(wc -l < git-checkpoint.sh)
    local covered_lines=0
    
    # Merge all trace files
    cat "$COVERAGE_DIR"/trace-*.txt | 
        grep -o '^+[^:]*:[0-9]*:' | 
        sed 's/+.*:\([0-9]*\):.*/\1/' | 
        sort -n | uniq > "$COVERAGE_DIR/covered-lines.txt"
    
    covered_lines=$(wc -l < "$COVERAGE_DIR/covered-lines.txt")
    local percentage=$((covered_lines * 100 / total_lines))
    
    # Generate HTML report
    cat > "$REPORT_FILE" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>git-checkpoint Coverage Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .summary { background: #f0f0f0; padding: 15px; border-radius: 5px; }
        .covered { background-color: #c8f7c5; }
        .uncovered { background-color: #ffcccc; }
        .line-number { color: #666; padding-right: 10px; }
        pre { background: white; padding: 10px; border: 1px solid #ddd; }
    </style>
</head>
<body>
    <h1>git-checkpoint Coverage Report</h1>
    <div class="summary">
        <h2>Summary</h2>
        <p>Total Lines: $total_lines</p>
        <p>Covered Lines: $covered_lines</p>
        <p>Coverage: $percentage%</p>
        <p>Generated: $(date)</p>
    </div>
    
    <h2>Line Coverage</h2>
    <pre>
EOF
    
    # Add line-by-line coverage
    while IFS= read -r line_num; do
        echo "<span class='line-number'>$line_num</span><span class='covered'>$(sed -n "${line_num}p" git-checkpoint.sh)</span>" >> "$REPORT_FILE"
    done < "$COVERAGE_DIR/covered-lines.txt"
    
    echo "</pre></body></html>" >> "$REPORT_FILE"
    
    echo "Coverage report generated: $REPORT_FILE"
    echo "Coverage: $covered_lines/$total_lines ($percentage%)"
}

# Main execution
main() {
    init_coverage
    
    local failed_tests=0
    local total_tests=0
    
    # Run all tests
    for test_file in "$TEST_DIR"/*.bats; do
        total_tests=$((total_tests + 1))
        if ! run_test_with_coverage "$test_file"; then
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Generate report
    generate_report
    
    # Summary
    echo -e "\n${GREEN}=== Test Summary ===${NC}"
    echo "Total tests: $total_tests"
    echo "Failed tests: $failed_tests"
    
    if [[ $failed_tests -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    fi
}

# Signal handling for cleanup
trap 'echo "Interrupted, saving coverage..."; generate_report; exit 130' INT TERM

# Run main
main
```

## 3. Function-Level Coverage Integration

Add minimal instrumentation without modifying core logic:

```bash
#!/bin/bash
# git-checkpoint-instrumented.sh - Minimal instrumentation

# Source the original script
source git-checkpoint.sh

# Override functions with coverage tracking
original_checkpoint_create=$(declare -f checkpoint_create)
checkpoint_create() {
    [[ "$COVERAGE_ENABLED" == "true" ]] && echo "FUNC:checkpoint_create" >&3
    eval "${original_checkpoint_create#*\{}"
}

original_checkpoint_restore=$(declare -f checkpoint_restore)
checkpoint_restore() {
    [[ "$COVERAGE_ENABLED" == "true" ]] && echo "FUNC:checkpoint_restore" >&3
    eval "${original_checkpoint_restore#*\{}"
}

# Continue for other functions...
```

## 4. MCPScriptTest Integration

Create a test that uses git-checkpoint with coverage:

```
# testdata/git_checkpoint_test.txt
# Test git-checkpoint with coverage

# Enable bash coverage
env MCP_BASH_COVERAGE=1
env MCP_BASH_COVERAGE_DIR=$WORK/coverage

# Initialize git repo
exec git init
exec git config user.email "test@example.com"
exec git config user.name "Test User"

# Create initial commit
exec echo "test" > file.txt
exec git add file.txt
exec git commit -m "Initial commit"

# Run git-checkpoint with coverage
bash './git-checkpoint.sh create "Test checkpoint"'
stdout 'Checkpoint created'

# List checkpoints
bash './git-checkpoint.sh list'
stdout 'Test checkpoint'

# Analyze coverage
exec cat $WORK/coverage/bash-*.trace
stdout 'checkpoint_create():'
stdout 'checkpoint_list():'

-- git-checkpoint.sh --
#!/bin/bash
# ... actual git-checkpoint script content ...
```

## 5. CI/CD Integration

GitHub Actions workflow with coverage:

```yaml
name: Test with Coverage

on: [push, pull_request]

jobs:
  test-coverage:
    runs-on: ubuntu-latest
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up test environment
      run: |
        sudo apt-get update
        sudo apt-get install -y bats
    
    - name: Run tests with coverage
      run: |
        chmod +x test-with-coverage.sh
        ./test-with-coverage.sh
      env:
        COVERAGE_ENABLED: true
    
    - name: Upload coverage report
      uses: actions/upload-artifact@v3
      with:
        name: coverage-report
        path: .coverage/report.html
    
    - name: Comment coverage on PR
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          const coverage = fs.readFileSync('.coverage/coverage-summary.txt', 'utf8');
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: '## Coverage Report\n```\n' + coverage + '\n```'
          });
```

## 6. Coverage Visualization

Create a visual coverage report:

```bash
#!/bin/bash
# visualize-coverage.sh

# Generate coverage heatmap
generate_heatmap() {
    local script_file=$1
    local coverage_dir=$2
    local output_file="${3:-coverage-heatmap.html}"
    
    # Count executions per line
    declare -A line_counts
    
    while IFS= read -r trace_line; do
        if [[ $trace_line =~ ^\+[^:]*:([0-9]+): ]]; then
            line_num="${BASH_REMATCH[1]}"
            line_counts[$line_num]=$((${line_counts[$line_num]:-0} + 1))
        fi
    done < <(cat "$coverage_dir"/trace-*.txt)
    
    # Find max count for scaling
    local max_count=0
    for count in "${line_counts[@]}"; do
        ((count > max_count)) && max_count=$count
    done
    
    # Generate HTML with heatmap
    cat > "$output_file" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Coverage Heatmap</title>
    <style>
        .line { font-family: monospace; white-space: pre; }
        .line-num { color: #666; padding-right: 10px; }
        .heat-0 { background: #fff; }
        .heat-1 { background: #fef0f0; }
        .heat-2 { background: #fde0e0; }
        .heat-3 { background: #fcc0c0; }
        .heat-4 { background: #fb8080; }
        .heat-5 { background: #fa4040; }
    </style>
</head>
<body>
    <h1>git-checkpoint Coverage Heatmap</h1>
    <p>Darker red = more executions</p>
    <pre>
EOF
    
    local line_num=1
    while IFS= read -r line; do
        local count=${line_counts[$line_num]:-0}
        local heat_level=0
        
        if ((count > 0)); then
            heat_level=$((count * 5 / max_count))
            ((heat_level > 5)) && heat_level=5
        fi
        
        printf '<span class="line"><span class="line-num">%4d</span><span class="heat-%d">%s</span></span>\n' \
            "$line_num" "$heat_level" "$line" >> "$output_file"
        
        ((line_num++))
    done < "$script_file"
    
    echo "</pre></body></html>" >> "$output_file"
    echo "Heatmap generated: $output_file"
}

# Generate for git-checkpoint
generate_heatmap "git-checkpoint.sh" ".coverage" "coverage-heatmap.html"
```

## Conclusion

This implementation provides:

1. **Non-invasive coverage**: No modifications to the original script
2. **Safe execution**: Resource limits and timeouts
3. **Comprehensive data**: Line, function, and execution count tracking
4. **Visual reports**: HTML reports and heatmaps
5. **CI/CD ready**: Automated testing and reporting
6. **MCPScriptTest integration**: Works with the testing framework

The key is to maintain the script's original behavior while collecting detailed execution data safely and efficiently.