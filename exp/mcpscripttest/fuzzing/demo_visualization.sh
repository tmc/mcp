#!/bin/bash

# Demo script for fuzzing visualization

echo "=== MCPScriptTest Fuzzing Visualization Demo ==="
echo
echo "This demo shows different visualization modes for fuzzing"
echo

# Basic visualization
echo "1. Basic visualization (shows only accepted scripts):"
echo "   go test -v -run TestVisualizationExample"
echo
read -p "Press Enter to continue..."
go test -v -run TestVisualizationExample

echo
echo "2. Fuzzing with visualization (use Ctrl+C to stop):"
echo "   MCP_FUZZ_VISUALIZE=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s"
echo
read -p "Press Enter to continue..."
MCP_FUZZ_VISUALIZE=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s

echo
echo "3. Fuzzing with rejected scripts shown:"
echo "   MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_SHOW_REJECTED=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s"
echo
read -p "Press Enter to continue..."
MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_SHOW_REJECTED=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s

echo
echo "4. Fuzzing with clear screen between updates:"
echo "   MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_CLEAR_SCREEN=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s"
echo
read -p "Press Enter to continue..."
MCP_FUZZ_VISUALIZE=1 MCP_FUZZ_CLEAR_SCREEN=1 go test -fuzz=FuzzSimpleServer -fuzztime=10s

echo
echo "Demo complete!"