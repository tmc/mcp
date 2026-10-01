#!/bin/bash
# Test JSON scanner functionality in mcpspy
#
# This script runs comprehensive tests for mcpspy's JSON scanning capabilities

# Expected to be run by TestJSONScanner in main_test.go
set -ex

# Create test input files if they don't exist
if [ ! -f testdata/input.json ]; then
    echo '{"method": "initialize", "params": {"protocolVersion": "2024-11-05", "clientInfo": {"name": "test-client", "version": "1.0.0"}, "capabilities": {}}}' > testdata/input.json
    echo '{"method": "tools/list", "params": {}}' >> testdata/input.json  
    echo '{"method": "tools/call", "params": {"name": "TestTool", "arguments": {"key": "value"}}}' >> testdata/input.json
fi

if [ ! -f testdata/split.json ]; then
    cat > testdata/split.json << 'EOF'
{"method": "split_test", "params": {
    "test": true,
    "value": "test value"
  }
}
EOF
fi

# Test 1: Basic JSON processing
cat testdata/input.json | mcpspy -f output1.log -v
grep -c "mcp-recv" output1.log || true
cat output1.log || true
grep -F '"method": "initialize"' output1.log
grep -F '"method": "tools/list"' output1.log
grep -F '"method": "tools/call"' output1.log

# Test 2: Pretty-print JSON option
cat testdata/input.json | mcpspy -f output2.log -v -pretty
grep -F '  "method": "initialize"' output2.log
grep -F '  "method": "tools/list"' output2.log
grep -F '  "method": "tools/call"' output2.log
grep -F '    "arguments": {' output2.log
grep -F '  "method":' output2.log

# Test 3: Split/fragmented JSON (simulate slow input)
cat testdata/split.json | mcpspy -f output3.log -v
grep -F '{"method": "split_test"' output3.log

echo "All tests passed!"
