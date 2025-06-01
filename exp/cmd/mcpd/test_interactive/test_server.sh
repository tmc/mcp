#!/bin/bash
set -euo pipefail

# Test script for running mcpspy with the everything server
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT=7000
LOG_FILE="${SCRIPT_DIR}/test_server.log"

# Export the command to run the MCP server
export SPYCMD="npx @modelcontextprotocol/server-everything stdio"

# Check dependencies
for cmd in socat mcpspy npx; do
  if ! command -v "$cmd" &> /dev/null; then
    echo "Error: $cmd is not installed or not in PATH" >&2
    exit 1
  fi
done

# Clean up any previous instances
cleanup() {
  echo "Cleaning up..." >&2
  
  # Find and kill any socat processes listening on our port
  local pids=$(lsof -i ":${PORT}" -t 2>/dev/null || true)
  if [[ -n "$pids" ]]; then
    echo "Killing previous socat instances on port ${PORT}..." >&2
    echo "$pids" | xargs kill -9 2>/dev/null || true
  fi
}

# Clean up on exit
trap cleanup EXIT

# First clean up any existing processes
cleanup

# Start the server
echo "Starting server on port ${PORT}..." >&2
echo "Logs will be written to ${LOG_FILE}" >&2
echo "Connect to the server using: nc localhost ${PORT}" >&2
echo "Press Ctrl+C to stop the server" >&2

# Run socat with mcpspy and log output
socat TCP-LISTEN:${PORT},fork,reuseaddr EXEC:"mcpspy -v -vv -- ${SPYCMD}" > "${LOG_FILE}" 2>&1 &
server_pid=$!

# Wait a moment to ensure the server is running
sleep 1

# Check if the server started successfully
if ! ps -p $server_pid > /dev/null; then
  echo "Error: Failed to start server process" >&2
  exit 1
fi

# Check if the port is open
if ! lsof -i ":${PORT}" > /dev/null 2>&1; then
  echo "Error: Failed to listen on port ${PORT}" >&2
  exit 1
fi

echo "Server started successfully with PID: ${server_pid}" >&2
echo "Tailing log file..." >&2

# Tail the log file
tail -f "${LOG_FILE}"