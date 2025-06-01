#!/bin/bash
# Script to test the jsonrpc2gostruct tool

set -e  # Exit on error

echo "Using pre-built jsonrpc2gostruct_bin..."
# Use the pre-built binary
TOOL="./jsonrpc2gostruct_bin"

# Create a temp directory for tests
TESTDIR=$(mktemp -d)
echo "Using test directory: $TESTDIR"

# Cleanup on exit
trap 'rm -rf "$TESTDIR"' EXIT

# --- Test 1: Basic Schema Conversion ---
echo "Test 1: Basic Schema Conversion"
cat > "$TESTDIR/schema.json" << 'EOF'
{
  "type": "object",
  "description": "A simple test schema",
  "properties": {
    "name": {
      "type": "string",
      "description": "The name of the item"
    },
    "count": {
      "type": "integer",
      "description": "The count of items"
    },
    "active": {
      "type": "boolean",
      "description": "Whether the item is active"
    }
  },
  "required": ["name", "active"]
}
EOF

OUTPUT=$($TOOL -package test "$TESTDIR/schema.json")
echo "$OUTPUT" > "$TESTDIR/output.go"

# Check for expected output
if ! grep -q "package test" "$TESTDIR/output.go"; then
    echo "FAIL: Missing package declaration"
    exit 1
fi

if ! grep -q "Schema - A simple test schema" "$TESTDIR/output.go"; then
    echo "FAIL: Missing schema description"
    exit 1
fi

if ! grep -q "Active bool \`json:\"active\"\`" "$TESTDIR/output.go"; then
    echo "FAIL: Missing Active field"
    exit 1
fi

if ! grep -q "Name string \`json:\"name\"\`" "$TESTDIR/output.go"; then
    echo "FAIL: Missing Name field"
    exit 1
fi

echo "Test 1: PASSED"

# --- Test 2: JSON-RPC Request Conversion ---
echo "Test 2: JSON-RPC Request Conversion"
cat > "$TESTDIR/jsonrpc.json" << 'EOF'
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "calculator",
    "arguments": {
      "operation": "add",
      "a": 5,
      "b": 3
    }
  }
}
EOF

OUTPUT=$($TOOL -package rpc "$TESTDIR/jsonrpc.json")
echo "$OUTPUT" > "$TESTDIR/rpc_output.go"

# Check for expected output
if ! grep -q "package rpc" "$TESTDIR/rpc_output.go"; then
    echo "FAIL: Missing package declaration"
    exit 1
fi

# Our converter treats JSON-RPC differently, look for method/params/id structure
if ! grep -q "Params" "$TESTDIR/rpc_output.go" || ! grep -q "Method" "$TESTDIR/rpc_output.go"; then
    echo "FAIL: Missing expected fields in JSON-RPC structure"
    cat "$TESTDIR/rpc_output.go"
    exit 1
fi

echo "Test 2: PASSED"

# --- Test 3: Date-Time Format Handling ---
echo "Test 3: Date-Time Format Handling"
cat > "$TESTDIR/format.json" << 'EOF'
{
  "type": "object",
  "description": "Schema with date-time format",
  "properties": {
    "created": {
      "type": "string",
      "format": "date-time",
      "description": "Creation timestamp"
    },
    "updated": {
      "type": "string",
      "format": "date-time",
      "description": "Last update timestamp"
    }
  }
}
EOF

OUTPUT=$($TOOL -package formats "$TESTDIR/format.json")
echo "$OUTPUT" > "$TESTDIR/format_output.go"

# Check for expected output
if ! grep -q "package formats" "$TESTDIR/format_output.go"; then
    echo "FAIL: Missing package declaration"
    exit 1
fi

if ! grep -q "import (" "$TESTDIR/format_output.go" && grep -q "\"time\"" "$TESTDIR/format_output.go"; then
    echo "FAIL: Missing time package import"
    exit 1
fi

if ! grep -q "Created time.Time" "$TESTDIR/format_output.go"; then
    echo "FAIL: Should use time.Time for date-time format"
    exit 1
fi

echo "Test 3: PASSED"

# --- Test 4: Tools Response Conversion ---
echo "Test 4: Tools Response Conversion"
cat > "$TESTDIR/tools.json" << 'EOF'
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "calculator",
        "description": "A simple calculator tool",
        "inputSchema": {
          "type": "object",
          "properties": {
            "operation": {
              "type": "string",
              "description": "The operation to perform"
            },
            "a": {
              "type": "number",
              "description": "First operand"
            },
            "b": {
              "type": "number",
              "description": "Second operand"
            }
          },
          "required": ["operation", "a", "b"]
        }
      }
    ]
  }
}
EOF

OUTPUT=$($TOOL -package tools "$TESTDIR/tools.json")
echo "$OUTPUT" > "$TESTDIR/tools_output.go"

# Check for expected output
if ! grep -q "package tools" "$TESTDIR/tools_output.go"; then
    echo "FAIL: Missing package declaration"
    exit 1
fi

if ! grep -q "CalculatorInput" "$TESTDIR/tools_output.go"; then
    echo "FAIL: Missing CalculatorInput struct"
    exit 1
fi

if ! grep -q "A float64" "$TESTDIR/tools_output.go"; then
    echo "FAIL: Missing proper type for A field"
    exit 1
fi

echo "Test 4: PASSED"

# --- Test 5: Output File Option ---
echo "Test 5: Output File Option"
$TOOL -package custom -out "$TESTDIR/custom_output.go" "$TESTDIR/schema.json"

# Check if file was created
if [ ! -f "$TESTDIR/custom_output.go" ]; then
    echo "FAIL: Output file was not created"
    exit 1
fi

if ! grep -q "package custom" "$TESTDIR/custom_output.go"; then
    echo "FAIL: Missing custom package name"
    exit 1
fi

echo "Test 5: PASSED"

# --- All tests passed ---
echo "All tests PASSED"