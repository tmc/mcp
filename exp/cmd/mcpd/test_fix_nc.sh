#!/bin/bash
# Test script for mcpd interactive mode with a server that doesn't exit immediately

# Use nc in listen mode instead of client mode
# -l makes nc listen instead of connect
# -k keeps listening after client disconnects
# -p specifies the port to listen on

# Build mcpd first
cd /Volumes/tmc/go/src/github.com/tmc/mcp/exp/cmd/mcpd
go build

# Run mcpd with nc in server mode
./mcpd -v -i -socket /tmp/mcpd-interactive-test.sock -- nc -l -k 8080

# This will keep the nc process running until you connect to it on port 8080
# You can test it with: echo "hello" | nc localhost 8080