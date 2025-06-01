#!/bin/bash

# Example script showing how to use cmd2mcpserver

echo "=== cmd2mcpserver Example ==="
echo

# Build demo CLI
echo "1. Building demo CLI..."
go build -o demo-cli ../demo/demo.go
echo "   ✓ Built demo-cli"
echo

# Convert to MCP server
echo "2. Converting to MCP server..."
go run ../../cmd/cmd2mcpserver -output ./demo-server -module github.com/example/demo-server -source ../demo ./demo-cli
echo

# Show generated files
echo "3. Generated files:"
find demo-server -type f | sort
echo

# Show generated server code
echo "4. Generated server code (first 50 lines):"
head -n 50 demo-server/main.go
echo

echo "=== To run the generated server ==="
echo "cd demo-server"
echo "go run ."
echo

echo "=== Demo complete ==="