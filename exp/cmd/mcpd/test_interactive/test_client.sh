#!/bin/bash
set -euo pipefail

# Test client script for connecting to the MCP server
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT=7000
LOG_FILE="${SCRIPT_DIR}/test_client.log"

# Check if nc (netcat) is available
if ! command -v nc &> /dev/null; then
  echo "Error: nc (netcat) is not installed or not in PATH" >&2
  exit 1
fi

# Ensure the server is running
if ! lsof -i ":${PORT}" > /dev/null 2>&1; then
  echo "Error: Server is not running on port ${PORT}" >&2
  echo "Please start the server first with: ./test_server.sh" >&2
  exit 1
fi

echo "Connecting to MCP server on localhost:${PORT}..." >&2
echo "Press Ctrl+C to disconnect" >&2
echo "Logging client interaction to ${LOG_FILE}" >&2

# Use script to capture the terminal session for later analysis
script -q -t 0 "${LOG_FILE}" nc localhost ${PORT}

echo "Client session ended. Log saved to ${LOG_FILE}" >&2