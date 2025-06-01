#!/bin/bash
# Test script for mcp-shadow

echo "Building mcp-shadow..."
go build .

echo -e "\n1. Basic shadow test:"
cat << EOF | ./mcp-shadow -primary "cat" -shadow "sed 's/test/shadow/g'" -o basic.mcp -q
{"jsonrpc":"2.0","method":"test.method","id":1}
{"jsonrpc":"2.0","id":1,"result":"test result"}
EOF

echo -e "\nContents of basic.mcp:"
cat basic.mcp

echo -e "\n2. Shadow test with trace context:"
cat << EOF | ./mcp-shadow -primary "cat" -shadow "jq '.shadow=true'" -trace -baggage "env=test" -o traced.mcp -q
{"jsonrpc":"2.0","method":"initialize","id":1}
{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}}
EOF

echo -e "\nContents of traced.mcp:"
cat traced.mcp

echo -e "\n3. Shadow with sampling (50%):"
cat << EOF | ./mcp-shadow -primary "cat" -shadow "jq '.sampled=true'" -split-mode random -split-percent 50 -o sampled.mcp -q
{"jsonrpc":"2.0","method":"request1","id":1}
{"jsonrpc":"2.0","method":"request2","id":2}
{"jsonrpc":"2.0","method":"request3","id":3}
EOF

echo -e "\nContents of sampled.mcp (some requests may not be shadowed):"
cat sampled.mcp

echo -e "\n4. Using with mcp-echo-server (if available):"
if command -v mcp-echo-server &> /dev/null; then
    cat << EOF | ./mcp-shadow -primary "mcp-echo-server" -shadow "mcp-echo-server --prefix shadow" -trace -o echo.mcp -q
{"jsonrpc":"2.0","method":"echo","params":{"message":"hello"},"id":1}
EOF
    echo -e "\nContents of echo.mcp:"
    cat echo.mcp
else
    echo "mcp-echo-server not found, skipping this test"
fi

echo -e "\nTest complete!"