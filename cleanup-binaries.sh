#!/bin/bash
# Script to remove binaries and temporary files before committing

echo "Removing binary executables..."

# Root directory binaries
rm -f mcp-probe mcp-replay mcp-shadow mcpdiff mcpspy cmd2mcpserver ctx-go-src mcp2go mcpscripttest
rm -f mcp.test mcp-test-Time-Server-Initialize modelcontextprotocol/generictypes.test

# Command directory binaries  
rm -f cmd/mcp-replay/mcp-replay
rm -f cmd/mcpdiff/mcpdiff
rm -f cmd/mcpspy/mcpspy

# Experimental tools binaries
rm -f exp/cov2codecov/cov2codecov
rm -f exp/covdiff/covdiff
rm -f exp/covtest/covtest
rm -f exp/mcpscripttest/stitch-demo

echo "Removing temporary and generated files..."

# Temporary files
rm -f cat_output1.txt cat_output2.txt
rm -f test_output.log
rm -f cmd/mcpspy/echo_test.log
rm -f vt-o x x-o

# Coverage files
find . -name "*.out" -type f -delete
rm -f coverage.html

# Demo files
rm -f codecov_demo.json demo_codecov.json coverage_test.json

echo "Removing coverage directories..."
rm -rf coverage_*/ coverage_analysis/ coverage_output/ coverage_demo/

echo "Removing temporary directories..."
rm -rf temp/ tmp/

echo "Cleanup complete! Review git status to confirm."