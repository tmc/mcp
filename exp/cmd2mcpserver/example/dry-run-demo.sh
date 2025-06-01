#!/bin/bash

# Demo script showing dry-run functionality

echo "=== cmd2mcpserver Dry-Run Demo ==="
echo 
echo "1. First, let's build our demo CLI:"
echo

cd ../demo
go build -o demo-cli demo.go
echo "✓ Built demo-cli"
echo

echo "2. Preview the generated server without creating files:"
echo

cd ../../cmd/cmd2mcpserver
echo "Running: go run . -dry-run -source ../../cmd2mcpserver/demo ../../cmd2mcpserver/demo/demo-cli"
echo

go run . -dry-run -source ../../cmd2mcpserver/demo ../../cmd2mcpserver/demo/demo-cli

echo
echo "=== Demo complete ==="
echo
echo "You can save this output to a file:"
echo "  go run . -dry-run -source ./demo ./demo-cli > server.txtar"
echo
echo "Or create the actual server:"
echo "  go run . -source ./demo ./demo-cli"