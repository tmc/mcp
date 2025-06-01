#!/bin/bash
# Test all MCP tools integration

set -e

echo "=== Testing MCP Tools Integration ==="

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
if mcpcat test1.mcp | grep -q 'mcp-recv' && mcpcat test1.mcp | grep -q 'mcp-send'; then
    echo "✓ mcpcat works"
else
    echo "✗ mcpcat failed"
    exit 1
fi

# Test with color disabled
if mcpcat -color=never test1.mcp | grep -q 'mcp-recv'; then
    echo "✓ mcpcat -color=never works"
fi

# 3. Test mcp-sort
echo "Testing mcp-sort..."
if mcp-sort -strip test1.mcp | grep -q '\[TIMESTAMP\]'; then
    echo "✓ mcp-sort -strip works"
else
    echo "✗ mcp-sort -strip failed"
    exit 1
fi

# Test sorting
mcp-sort test1.mcp > sorted.mcp
if [ -s sorted.mcp ]; then
    echo "✓ mcp-sort works"
fi

# 4. Test mcpdiff
echo "Testing mcpdiff..."
cp test1.mcp test2.mcp
echo 'mcp-recv {"jsonrpc":"2.0","method":"new","id":3} # 1004.000' >> test2.mcp

# mcpdiff should return exit code 1 for differences
if ! mcpdiff test1.mcp test2.mcp > diff.txt && grep -q 'new' diff.txt; then
    echo "✓ mcpdiff works"
else
    echo "✗ mcpdiff failed"
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
if mcpdiff -compare shadow.mcp > compare.txt && grep -q 'tools' compare.txt; then
    echo "✓ mcpdiff -compare works"
else
    echo "✗ mcpdiff -compare failed"
    exit 1
fi

# Test mcpcat with shadow traces
if mcpcat shadow.mcp | grep -q 'mcp-send-shadow'; then
    echo "✓ mcpcat handles shadow traces"
fi

# 6. Test mcpspy
echo "Testing mcpspy..."
echo "test data" > input.txt
if mcpspy -f spy.mcp cat input.txt | grep -q 'test data' && [ -f spy.mcp ]; then
    echo "✓ mcpspy works"
else
    echo "✗ mcpspy failed"
    exit 1
fi

# 7. Test tool chain
echo "Testing tool chain..."
echo '{"jsonrpc":"2.0","method":"test","id":1}' | mcpspy -f chain.mcp cat > /dev/null
if [ -f chain.mcp ]; then
    mcpcat chain.mcp | mcp-sort -strip > chain_processed.mcp
    if [ -s chain_processed.mcp ]; then
        echo "✓ Tool chain works"
    fi
fi

echo "=== All tests passed! ==="

# Cleanup
cd ..
rm -rf $TEST_DIR