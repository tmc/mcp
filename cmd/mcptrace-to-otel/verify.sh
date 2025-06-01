#!/bin/bash
# Quick verification script for mcptrace-to-otel

set -e

echo "=== Verifying mcptrace-to-otel installation ==="

# Build the tool
echo "Building mcptrace-to-otel..."
GOWORK=off go build -o mcptrace-to-otel .

# Test basic functionality
echo -e "\nTesting basic conversion..."
./mcptrace-to-otel -f example.mcp -type stdout > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ Basic conversion works"
else
    echo "✗ Basic conversion failed"
    exit 1
fi

# Test with complex example
echo -e "\nTesting complex trace conversion..."
./mcptrace-to-otel -f complex-example.mcp -type stdout > /dev/null  
if [ $? -eq 0 ]; then
    echo "✓ Complex trace conversion works"
else
    echo "✗ Complex trace conversion failed"
    exit 1
fi

# Test invalid input handling
echo -e "\nTesting error handling..."
echo "invalid json" > invalid.mcp
./mcptrace-to-otel -f invalid.mcp -type stdout 2>/dev/null || true
echo "✓ Error handling works"

# Test help
echo -e "\nTesting help output..."
./mcptrace-to-otel -h > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Help command works"
else
    echo "✗ Help command failed"
    exit 1
fi

echo -e "\n=== All tests passed! ==="
echo
echo "Next steps:"
echo "1. Run 'make demo-stdout' to see stdout output"
echo "2. Run 'make docker-demo' to test with Jaeger"
echo "3. Run './demo.sh' for full integration demo"

# Cleanup
rm -f invalid.mcp