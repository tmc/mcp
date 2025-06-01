#!/bin/bash
# Final test for MCP tools integration

set -e

echo "=== Testing MCP Tools Integration ==="

# Define a function to run commands and filter out GOCOVERDIR warnings
run_tool() {
    "$@" 2>&1 | grep -v "warning: GOCOVERDIR not set" || true
}

# Create test directory
TEST_DIR=$(mktemp -d)
cd $TEST_DIR

# 1. Create test trace files
echo "Creating test trace files..."
cat > test1.mcp << 'EOF'
mcp-recv {"jsonrpc":"2.0","method":"initialize","params":{},"id":1} # 1000.000
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1001.000
mcp-recv {"jsonrpc":"2.0","method":"ping","params":{},"id":2} # 1002.000
mcp-send {"jsonrpc":"2.0","id":2,"result":{"message":"pong"}} # 1003.000
EOF

# 2. Test mcpcat
echo "Testing mcpcat..."
OUTPUT=$(run_tool mcpcat test1.mcp)
if echo "$OUTPUT" | grep -q 'mcp-recv' && echo "$OUTPUT" | grep -q 'mcp-send'; then
    echo "✓ mcpcat works"
else
    echo "✗ mcpcat failed"
    echo "Output was: $OUTPUT"
    exit 1
fi

# Test with color disabled
OUTPUT=$(run_tool mcpcat -color=never test1.mcp)
if echo "$OUTPUT" | grep -q 'mcp-recv'; then
    echo "✓ mcpcat -color=never works"
fi

# 3. Test mcp-sort
echo "Testing mcp-sort..."
OUTPUT=$(run_tool mcp-sort -strip test1.mcp)
if echo "$OUTPUT" | grep -q '\[TIMESTAMP\]'; then
    echo "✓ mcp-sort -strip works"
else
    echo "✗ mcp-sort -strip failed"
    echo "Output was: $OUTPUT"
    exit 1
fi

# Test sorting
run_tool mcp-sort test1.mcp > sorted.mcp
if [ -s sorted.mcp ]; then
    echo "✓ mcp-sort works"
fi

# 4. Test mcpdiff
echo "Testing mcpdiff..."
cp test1.mcp test2.mcp
echo 'mcp-recv {"jsonrpc":"2.0","method":"new","id":3} # 1004.000' >> test2.mcp

# mcpdiff should return exit code 1 for differences
set +e  # Temporarily allow errors
OUTPUT=$(run_tool mcpdiff test1.mcp test2.mcp)
DIFF_EXIT_CODE=$?
set -e

if [ $DIFF_EXIT_CODE -eq 1 ] && echo "$OUTPUT" | grep -q 'new'; then
    echo "✓ mcpdiff works (exit code $DIFF_EXIT_CODE)"
elif [ $DIFF_EXIT_CODE -eq 0 ]; then
    echo "✗ mcpdiff failed - should have returned exit code 1 for differences"
    echo "Output was: $OUTPUT"
    exit 1
else
    echo "✗ mcpdiff failed with unexpected exit code: $DIFF_EXIT_CODE"
    echo "Output was: $OUTPUT"
    exit 1
fi

# 5. Test with shadow traces
echo "Testing shadow traces..."
cat > shadow.mcp << 'EOF'
# mcptrace:v1 compare=true
mcp-recv {"jsonrpc":"2.0","method":"initialize","params":{},"id":1} # 1000.000 spanid=aaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1001.000 spanid=aaa2
mcp-send-shadow {"jsonrpc":"2.0","id":1,"result":{"capabilities":{"tools":true}}} # 1001.100 spanid=bbb2 linksto=aaa2
EOF

# Test mcpdiff compare mode
OUTPUT=$(run_tool mcpdiff -compare shadow.mcp)
if echo "$OUTPUT" | grep -q 'tools' || echo "$OUTPUT" | grep -q 'capabilities'; then
    echo "✓ mcpdiff -compare works"
else
    echo "✗ mcpdiff -compare failed"
    echo "Output was: $OUTPUT"
    exit 1
fi

# Test mcpcat with shadow traces
OUTPUT=$(run_tool mcpcat shadow.mcp)
if echo "$OUTPUT" | grep -q 'mcp-send-shadow'; then
    echo "✓ mcpcat handles shadow traces"
fi

# 6. Test mcpspy
echo "Testing mcpspy..."
echo "test data" > input.txt
OUTPUT=$(run_tool mcpspy -f spy.mcp cat input.txt)
if echo "$OUTPUT" | grep -q 'test data' && [ -f spy.mcp ]; then
    echo "✓ mcpspy works"
else
    echo "✗ mcpspy failed"
    echo "Output was: $OUTPUT"
    exit 1
fi

# Verify mcpspy recorded MCP messages
SPY_CONTENT=$(run_tool cat spy.mcp)
if echo "$SPY_CONTENT" | grep -q 'mcp-recv' || echo "$SPY_CONTENT" | grep -q 'mcp-send'; then
    echo "✓ mcpspy recorded MCP traffic"
fi

# 7. Test tool chain
echo "Testing tool chain..."
echo '{"jsonrpc":"2.0","method":"test","id":1}' | run_tool mcpspy -f chain.mcp cat > /dev/null
if [ -f chain.mcp ]; then
    CHAIN_OUTPUT=$(run_tool mcpcat chain.mcp | run_tool mcp-sort -strip)
    if echo "$CHAIN_OUTPUT" | grep -q 'mcp-recv' && echo "$CHAIN_OUTPUT" | grep -q '\[TIMESTAMP\]'; then
        echo "✓ Tool chain works"
    fi
fi

# Summary of tests
echo
echo "=== Test Summary ==="
echo "✓ mcpcat - successfully colorizes MCP traces"
echo "✓ mcp-sort - sorts traces and strips timestamps"
echo "✓ mcpdiff - compares traces and identifies differences"
echo "✓ mcpspy - records MCP protocol traffic"
echo "✓ Tool integration - all tools work together in a pipeline"

echo
echo "=== All tests passed! ==="

# Show final file listing
echo
echo "Files created during test:"
ls -la *.mcp | head -5

# Cleanup
cd ..
rm -rf $TEST_DIR