#!/bin/bash
# Example: Using mcp-trace-codegen in a pipeline

echo "=== MCP Tool Pipeline Example ==="
echo

# Example 1: Capture live traffic and generate code
echo "Example 1: Live capture to code generation"
echo "mcpspy -- some-mcp-server | mcp-trace-codegen -realtime -package captured"
echo

# Example 2: Compare two traces and generate difference code  
echo "Example 2: Diff traces and generate code for differences"
echo "mcpdiff trace1.mcp trace2.mcp | mcp-trace-codegen -package diff_impl"
echo

# Example 3: Filter specific methods and generate code
echo "Example 3: Generate code only for tool calls"
echo "grep 'tools/' trace.mcp | mcp-trace-codegen -package tools_only"
echo

# Example 4: Real demo - monitor mcpd and generate code
echo "Example 4: Monitor mcpd daemon and generate code"
echo "Starting mock server with monitoring..."

# Create a mock trace that simulates mcpd output
(
echo "2024-01-15T10:00:00 -> initialize {\"method\":\"initialize\",\"params\":{\"protocolVersion\":\"1.0\",\"clientInfo\":{\"name\":\"mcpd-monitor\",\"version\":\"1.0\"}}}"
sleep 1
echo "2024-01-15T10:00:01 <- initialize {\"result\":{\"protocolVersion\":\"1.0\",\"serverInfo\":{\"name\":\"monitored-server\",\"version\":\"2.0\"}}}"
sleep 1
echo "2024-01-15T10:00:02 -> tools/list {\"method\":\"tools/list\",\"params\":{}}"
sleep 1
echo "2024-01-15T10:00:03 <- tools/list {\"result\":{\"tools\":[{\"name\":\"analyze_logs\",\"description\":\"Analyze log files\",\"inputSchema\":{\"type\":\"object\",\"properties\":{\"path\":{\"type\":\"string\"},\"pattern\":{\"type\":\"string\"}},\"required\":[\"path\"]}}]}}"
sleep 2
echo "2024-01-15T10:00:05 -> tools/call {\"method\":\"tools/call\",\"params\":{\"name\":\"analyze_logs\",\"arguments\":{\"path\":\"/var/log/app.log\",\"pattern\":\"ERROR\"}}}"
sleep 1
echo "2024-01-15T10:00:06 <- tools/call {\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"Found 3 ERROR entries in log file\"}]}}"
) | mcp-trace-codegen -realtime -clear -package log_analyzer

echo
echo "Pipeline example complete!"