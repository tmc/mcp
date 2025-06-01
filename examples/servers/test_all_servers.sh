#!/bin/bash

# Test automation script for all MCP servers
# This script builds each server and runs comprehensive tests

set -e

echo "=== Building and Testing MCP Servers ==="

# Function to test a server
test_server() {
    local server_dir=$1
    local server_name=$(basename "$server_dir")
    
    echo ""
    echo "Testing $server_name..."
    echo "=============================="
    
    cd "$server_dir"
    
    # Build the server
    echo "Building $server_name..."
    go build -o "$server_name" .
    
    # Run tests
    echo "Running tests for $server_name..."
    go test -v .
    
    echo "$server_name tests completed."
    cd - > /dev/null
}

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Test each server
test_server "$SCRIPT_DIR/mcp-time-server"
test_server "$SCRIPT_DIR/mcp-echo-server"
test_server "$SCRIPT_DIR/mcp-weather-server"

echo ""
echo "=== All server tests completed ==="