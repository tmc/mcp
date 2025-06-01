#!/bin/bash
set -euo pipefail

# Test script for sending signals to mcpd to test signal handling
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT=7000
MCPD_ARGS=""

usage() {
  echo "Usage: $0 [options]" >&2
  echo "" >&2
  echo "Options:" >&2
  echo "  -i, --interactive     Run mcpd in interactive mode (default)" >&2
  echo "  -n, --non-interactive Run mcpd in non-interactive mode" >&2
  echo "  -p, --port PORT       Port to connect to (default: 7000)" >&2
  echo "  -h, --help            Show this help message" >&2
  exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--interactive)
      MCPD_ARGS="--interactive"
      shift
      ;;
    -n|--non-interactive)
      MCPD_ARGS=""
      shift
      ;;
    -p|--port)
      if [[ -n "${2:-}" ]]; then
        PORT="$2"
        shift 2
      else
        echo "Error: Port number is required for --port option" >&2
        usage
      fi
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Error: Unknown option: $1" >&2
      usage
      ;;
  esac
done

# Check if the server is running
if ! lsof -i ":${PORT}" > /dev/null 2>&1; then
  echo "Error: Server is not running on port ${PORT}" >&2
  echo "Please start the server first with: ./test_server.sh" >&2
  exit 1
fi

# Build and run mcpd with specified options
echo "Building mcpd..." >&2
(cd .. && go build)

echo "Starting mcpd in the background..." >&2
../mcpd ${MCPD_ARGS} "localhost:${PORT}" > "${SCRIPT_DIR}/mcpd.log" 2>&1 &
MCPD_PID=$!

# Wait for mcpd to start
sleep 2

if ! ps -p $MCPD_PID > /dev/null; then
  echo "Error: mcpd failed to start" >&2
  cat "${SCRIPT_DIR}/mcpd.log"
  exit 1
fi

echo "mcpd started with PID: ${MCPD_PID}" >&2
echo "Log file: ${SCRIPT_DIR}/mcpd.log" >&2

echo "Available signal tests:"
echo "1. SIGWINCH - Terminal window resize"
echo "2. SIGTSTP - Terminal suspend (Ctrl+Z)"
echo "3. SIGCONT - Continue after suspend"
echo "4. SIGINT - Interrupt (Ctrl+C)"
echo "5. SIGTERM - Termination signal"
echo "6. SIGHUP - Hangup signal"
echo "7. SIGTTIN - Terminal read from background"
echo "8. SIGTTOU - Terminal write to background"
echo "9. Kill mcpd"
echo "q. Quit this script"

# Function to send a signal to mcpd
send_signal() {
  local signal="$1"
  echo "Sending ${signal} to mcpd (PID: ${MCPD_PID})..." >&2
  kill -s "${signal}" "${MCPD_PID}" 2>/dev/null || echo "Failed to send signal (process may have terminated)"
  sleep 1
  if ps -p "${MCPD_PID}" > /dev/null; then
    echo "mcpd is still running" >&2
  else
    echo "mcpd has terminated" >&2
  fi
}

# Main interactive loop
while true; do
  echo ""
  read -p "Enter test number (q to quit): " choice
  
  case "$choice" in
    1)
      send_signal "WINCH"
      ;;
    2)
      send_signal "TSTP"
      ;;
    3)
      send_signal "CONT"
      ;;
    4)
      send_signal "INT"
      ;;
    5)
      send_signal "TERM"
      ;;
    6)
      send_signal "HUP"
      ;;
    7)
      send_signal "TTIN"
      ;;
    8)
      send_signal "TTOU"
      ;;
    9)
      echo "Killing mcpd..." >&2
      kill -9 "${MCPD_PID}" 2>/dev/null || echo "mcpd is not running"
      break
      ;;
    q|Q)
      echo "Quitting and cleaning up..." >&2
      if ps -p "${MCPD_PID}" > /dev/null; then
        kill "${MCPD_PID}" 2>/dev/null || true
      fi
      break
      ;;
    *)
      echo "Invalid choice" >&2
      ;;
  esac
  
  # Display the last few lines of the log to see the effect
  echo "Latest log entries:"
  tail -n 10 "${SCRIPT_DIR}/mcpd.log"
done

echo "Test completed" >&2