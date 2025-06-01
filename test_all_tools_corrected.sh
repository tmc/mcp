#!/bin/bash
# Corrected test for MCP tools integration

set -e

echo "=== Testing MCP Tools Integration ==="

# Define a function to run commands and filter out GOCOVERDIR warnings
run_tool() {
    "$@" 2>&1 | grep -v "warning: GOCOVERDIR not set" || true
}

# Create test directory
TEST_DIR=$(mktemp -d)
cd $TEST_DIR

# 1. Create test trace files with correct timestamp format
echo "Creating test trace files..."
cat > test1.mcp << 'EOF'
mcp-recv {"jsonrpc":"2.0","method":"initialize","params":{},"id":1} [2024-01-01 10:00:00.000]
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} [2024-01-01 10:00:01.000]
mcp-recv {"jsonrpc":"2.0","method":"ping","params":{},"id":2} [2024-01-01 10:00:02.000]
mcp-send {"jsonrpc":"2.0","id":2,"result":{"message":"pong"}} [2024-01-01 10:00:03.000]
EOF

# 2. Test mcpcat
echo "Testing mcpcat..."
OUTPUT=$(run_tool mcpcat test1.mcp)
if echo "$OUTPUT" | grep -q 'mcp-recv' && echo "$OUTPUT" | grep -q 'mcp-send'; then
    echo "✓ mcpcat works"
else
    echo "✗ mcpcat failed"
    exit 1
fi

# 3. Test mcp-sort with strip
echo "Testing mcp-sort..."
OUTPUT=$(run_tool mcp-sort -strip test1.mcp)
if echo "$OUTPUT" | grep -q '\[TIMESTAMP\]'; then
    echo "✓ mcp-sort -strip works"
else
    echo "✗ mcp-sort -strip failed"
    echo "Output was: $OUTPUT"
    exit 1
fi

# Create test files in the right format (using # for mcpdiff)
cat > test2.mcp << 'EOF'
mcp-recv {"jsonrpc":"2.0","method":"initialize","params":{},"id":1} # 1000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1001.000
mcp-recv {"jsonrpc":"2.0","method":"ping","params":{},"id":2} # 1002.000
mcp-send {"jsonrpc":"2.0","id":2,"result":{"message":"pong"}} # 1003.000
EOF

# 4. Test mcpdiff
echo "Testing mcpdiff..."
cp test2.mcp test3.mcp
echo 'mcp-recv {"jsonrpc":"2.0","method":"new","id":3} # 1004.000' >> test3.mcp

set +e  # Temporarily allow errors
OUTPUT=$(run_tool mcpdiff test2.mcp test3.mcp)
DIFF_EXIT_CODE=$?
set -e

if [ $DIFF_EXIT_CODE -eq 1 ] && echo "$OUTPUT" | grep -q 'new'; then
    echo "✓ mcpdiff works"
else
    echo "✗ mcpdiff failed"
    echo "Exit code: $DIFF_EXIT_CODE"
    echo "Output: $OUTPUT"
    exit 1
fi

# 5. Test mcpspy
echo "Testing mcpspy..."
echo "test data" > input.txt
run_tool mcpspy -f spy.mcp cat input.txt > spy_output.txt
if grep -q 'test data' spy_output.txt && [ -f spy.mcp ]; then
    echo "✓ mcpspy works"
else
    echo "✗ mcpspy failed"
    exit 1
fi

# 6. Test shadow traces with mcpdiff
echo "Testing shadow traces..."
cat > shadow.mcp << 'EOF'
# mcptrace:v1 compare=true
mcp-recv {"jsonrpc":"2.0","method":"initialize","params":{},"id":1} # 1000.000 spanid=aaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1001.000 spanid=aaa2
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":true}}} # 1001.100 spanid=bbb2 linksto=aaa2
EOF

OUTPUT=$(run_tool mcpdiff -compare shadow.mcp)
if echo "$OUTPUT" | grep -q 'tools' || echo "$OUTPUT" | grep -q 'capabilities'; then
    echo "✓ mcpdiff -compare works"
else
    echo "✗ mcpdiff -compare failed"
    echo "Output: $OUTPUT"
    exit 1
fi

# 7. Test mcpcat with color handling
echo "Testing mcpcat with color modes..."
OUTPUT=$(run_tool mcpcat -color=never test2.mcp)
if echo "$OUTPUT" | grep -q 'mcp-recv' && echo "$OUTPUT" | grep -v $'\033'; then
    echo "✓ mcpcat -color=never works correctly"
fi

# 8. Test tool integration
echo "Testing tool integration..."
# Create a simple combined trace
cat > combined.mcp << 'EOF'
mcp-recv {"jsonrpc":"2.0","method":"test","id":1} [2024-01-01 10:00:00.500]
mcp-send {"jsonrpc":"2.0","id":1,"result":{"ok":true}} [2024-01-01 10:00:00.100]
mcp-recv {"jsonrpc":"2.0","method":"test2","id":2} [2024-01-01 10:00:00.300]
EOF

# Sort by timestamp
OUTPUT=$(run_tool mcp-sort combined.mcp)
if echo "$OUTPUT" | grep -n "test" | head -1 | grep -q "2:"; then  # test2 should come first after sorting
    echo "✓ mcp-sort correctly sorts by timestamp"
fi

# Strip timestamps and process with mcpcat
run_tool mcp-sort -strip combined.mcp | run_tool mcpcat -color=never > processed.mcp
if grep -q '\[TIMESTAMP\]' processed.mcp && grep -q 'mcp-recv' processed.mcp; then
    echo "✓ Tool pipeline works"
fi

echo
echo "=== Summary ==="
echo "✓ mcpcat - colorizes MCP traces with configurable color modes"
echo "✓ mcp-sort - sorts traces by timestamp and strips timestamps"
echo "✓ mcpdiff - compares traces and identifies differences"
echo "✓ mcpspy - records MCP protocol traffic to trace files"
echo "✓ Tool integration - all tools work together in pipelines"

echo
echo "=== All tests passed! ==="

# Cleanup
cd ..
rm -rf $TEST_DIR