#!/bin/bash

# Automated test suite for MCP server capabilities

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_RESULTS_FILE="test_results.json"
LOG_FILE="test_log.txt"

# Initialize test results
echo '{"test_runs": []}' > "$TEST_RESULTS_FILE"

# Function to run a test round
run_test_round() {
    local round_num=$1
    local test_workspace="/tmp/mcp-test-round-$round_num"
    
    echo -e "${BLUE}=== Starting Test Round $round_num ===${NC}"
    
    # Create test workspace
    mkdir -p "$test_workspace"
    
    # Start server
    ./mcp-serve --workspace="$test_workspace" -- npx @modelcontextprotocol/server-everything stdio &
    local server_pid=$!
    
    # Wait for server to start
    sleep 3
    
    # Check if server is running
    if ! ./mcp-serve --workspace="$test_workspace" --status; then
        echo -e "${RED}Server failed to start${NC}"
        return 1
    fi
    
    # Run test sequence
    local test_count=0
    local pass_count=0
    local fail_count=0
    
    # Test 1: Initialize
    echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"TestClient","version":"1.0.0"}}}' | \
        ./mcp-serve --workspace="$test_workspace" --send > "$test_workspace/init.json" 2>/dev/null
    test_count=$((test_count + 1))
    if grep -q '"serverInfo"' "$test_workspace/init.json"; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    # Test 2: List prompts
    echo '{"jsonrpc":"2.0","id":2,"method":"prompts/list","params":{}}' | \
        ./mcp-serve --workspace="$test_workspace" --send > "$test_workspace/prompts.json" 2>/dev/null
    test_count=$((test_count + 1))
    if grep -q '"prompts"' "$test_workspace/prompts.json"; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    # Test 3: List tools
    echo '{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}' | \
        ./mcp-serve --workspace="$test_workspace" --send > "$test_workspace/tools.json" 2>/dev/null
    test_count=$((test_count + 1))
    if grep -q '"tools"' "$test_workspace/tools.json"; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    # Test 4: Call echo tool
    echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"echo","arguments":{"message":"Test Round '$round_num'"}}}' | \
        ./mcp-serve --workspace="$test_workspace" --send > "$test_workspace/echo.json" 2>/dev/null
    test_count=$((test_count + 1))
    if grep -q "Test Round $round_num" "$test_workspace/echo.json"; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    # Test 5: List resources
    echo '{"jsonrpc":"2.0","id":5,"method":"resources/list","params":{}}' | \
        ./mcp-serve --workspace="$test_workspace" --send > "$test_workspace/resources.json" 2>/dev/null
    test_count=$((test_count + 1))
    if grep -q '"resources"' "$test_workspace/resources.json"; then
        pass_count=$((pass_count + 1))
    else
        fail_count=$((fail_count + 1))
    fi
    
    # Stop server
    ./mcp-serve --workspace="$test_workspace" --stop
    
    # Record results
    local end_time=$(date +%s)
    local result_entry=$(cat <<EOF
{
    "round": $round_num,
    "timestamp": "$end_time",
    "tests": $test_count,
    "passed": $pass_count,
    "failed": $fail_count,
    "success_rate": $(echo "scale=2; $pass_count * 100 / $test_count" | bc)
}
EOF
)
    
    # Update results file
    jq --argjson entry "$result_entry" '.test_runs += [$entry]' "$TEST_RESULTS_FILE" > tmp.$$.json && mv tmp.$$.json "$TEST_RESULTS_FILE"
    
    echo -e "${BLUE}Round $round_num: ${GREEN}$pass_count passed${NC}, ${RED}$fail_count failed${NC}"
    
    # Clean up
    rm -rf "$test_workspace"
}

# Main test loop
echo "Building mcp-serve..."
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/mcp-serve
go build -o mcp-serve

echo -e "${GREEN}Starting automated test suite${NC}"

# Run multiple rounds
NUM_ROUNDS=5
for i in $(seq 1 $NUM_ROUNDS); do
    run_test_round $i
    sleep 2  # Brief pause between rounds
done

# Generate summary report
echo ""
echo -e "${BLUE}=== Test Summary ===${NC}"
jq -r '.test_runs[] | "Round \(.round): \(.passed)/\(.tests) passed (\(.success_rate)%)"' "$TEST_RESULTS_FILE"

echo ""
echo -e "${BLUE}=== Overall Statistics ===${NC}"
jq -r '
    .test_runs | 
    map(.tests) as $tests | 
    map(.passed) as $passed | 
    map(.failed) as $failed | 
    {
        total_tests: ($tests | add),
        total_passed: ($passed | add),
        total_failed: ($failed | add),
        overall_success_rate: (($passed | add) * 100.0 / ($tests | add))
    } | 
    "Total tests: \(.total_tests)\nPassed: \(.total_passed)\nFailed: \(.total_failed)\nSuccess rate: \(.overall_success_rate | round)%"
' "$TEST_RESULTS_FILE"

echo ""
echo "Detailed results saved to: $TEST_RESULTS_FILE"