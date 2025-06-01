#!/bin/bash
set -e

# Define cleanup function to ensure we always cleanup on exit
cleanup() {
    # Kill mcpd if it's running
    if [ -n "$MCPD_PID" ] && kill -0 $MCPD_PID 2>/dev/null; then
        echo "Stopping mcpd (PID: $MCPD_PID)..."
        kill -TERM $MCPD_PID 2>/dev/null || true

        # Give it a moment to clean up
        sleep 1

        # Force kill if still running
        if kill -0 $MCPD_PID 2>/dev/null; then
            kill -KILL $MCPD_PID 2>/dev/null || true
        fi
    fi

    # Kill the dummy server if it's running
    if [ -n "$SERVER_PID" ] && kill -0 $SERVER_PID 2>/dev/null; then
        echo "Stopping test server (PID: $SERVER_PID)..."
        kill -TERM $SERVER_PID 2>/dev/null || true
        sleep 1
        if kill -0 $SERVER_PID 2>/dev/null; then
            kill -KILL $SERVER_PID 2>/dev/null || true
        fi
    fi

    # Clean up socket and trace file
    rm -f "$SOCKET_PATH" "$TRACE_FILE"

    echo "Cleanup completed"
}

# Handle signals
trap cleanup EXIT INT TERM

# Build the test server
echo "Building test server..."
cd "$(dirname "$0")"
go build -o test_server server.go

# Build the always running server
echo "Building always running server..."
go build -o always_running_server always_running_server.go

# Build mcpd
echo "Building mcpd..."
cd ..
go build

# Path to mcpd and socket
MCPD_BIN="./mcpd"
SERVER_BIN="./test_interactive/test_server"
ALWAYS_RUNNING_SERVER_BIN="./test_interactive/always_running_server"
SOCKET_PATH="/tmp/mcpd-interactive-test.sock"
TRACE_FILE="/tmp/mcpd-interactive-test.mcp"

# Clean up socket and trace file if they exist
rm -f "$SOCKET_PATH" "$TRACE_FILE"

# Default to regular server
USE_ALWAYS_RUNNING=0

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
        --always-running)
            USE_ALWAYS_RUNNING=1
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--always-running]"
            exit 1
            ;;
    esac
done

# Start the appropriate server
if [ $USE_ALWAYS_RUNNING -eq 1 ]; then
    echo "Starting always running server..."
    "$ALWAYS_RUNNING_SERVER_BIN" &
    SERVER_PID=$!
    SERVER_ARGS="-socket $SOCKET_PATH"
else
    SERVER_PID=""
    SERVER_ARGS="$SERVER_BIN interactive_test"
fi

# Start mcpd with interactive mode
echo "Starting mcpd in interactive mode..."
"$MCPD_BIN" -i -socket "$SOCKET_PATH" -log-file "$TRACE_FILE" \
  -v -- $SERVER_ARGS &
MCPD_PID=$!

# Wait for mcpd to start and create socket
echo "Waiting for socket to be created..."
for i in {1..20}; do
    if [ -S "$SOCKET_PATH" ]; then
        break
    fi
    sleep 0.5
    if [ $i -eq 20 ]; then
        echo "Error: Socket not created after 10 seconds"
        exit 1
    fi
done

echo "Socket created: $SOCKET_PATH"
echo "MCPD running with PID: $MCPD_PID"

# Function to send a request and display response
send_request() {
    local method=$1
    local params=$2
    local timeout=${3:-5}

    echo "Sending request: $method"
    echo "{\"jsonrpc\":\"2.0\",\"id\":\"test-$(date +%s)\",\"method\":\"$method\",\"params\":$params}" | \
        timeout $timeout nc -U "$SOCKET_PATH"
    local status=$?
    echo

    if [ $status -ne 0 ]; then
        echo "Error: Request timed out or failed with status $status"
        return 1
    fi

    return 0
}

# Verify the server is responding
echo "Testing basic echo method..."
if ! send_request "echo" "{\"message\":\"Hello, World!\"}"; then
    echo "ERROR: Basic echo test failed - server not responding"
    exit 1
fi

echo "=== Interactive tests ==="
echo "NOTE: These require manual input. The script will continue automatically"
echo "      after you respond to the prompts."
echo

# Test prompt method
echo "Testing prompt method..."
echo "When prompted, enter a test name and press Enter"
send_request "prompt" "{\"message\":\"Please enter your name:\"}"

# Test prompt chain
echo "Testing prompt chain..."
echo "This will prompt you 3 times in sequence. Enter y/yes each time to continue."
send_request "prompt_chain" "{}"

# Test another echo to verify server is still working
echo "Testing echo method again to verify server is still operating..."
send_request "echo" "{\"message\":\"Hello, World after prompts!\"}"

# If user selects to test signal handling
echo
echo "=== Signal handling tests ==="
echo "Would you like to test signal handling? (y/n)"
read -r test_signals

if [[ "$test_signals" =~ ^[Yy] ]]; then
    echo "Testing SIGWINCH (terminal resize)..."
    echo "Sending SIGWINCH to mcpd (PID: $MCPD_PID)..."
    kill -WINCH $MCPD_PID
    sleep 1
    
    echo "Testing echo after SIGWINCH..."
    send_request "echo" "{\"message\":\"After SIGWINCH\"}"
    
    echo "Testing SIGTSTP (terminal suspend) and SIGCONT..."
    echo "Sending SIGTSTP to mcpd (PID: $MCPD_PID)..."
    kill -TSTP $MCPD_PID
    sleep 2
    
    echo "Sending SIGCONT to mcpd (PID: $MCPD_PID)..."
    kill -CONT $MCPD_PID
    sleep 1
    
    echo "Testing echo after SIGTSTP/SIGCONT..."
    send_request "echo" "{\"message\":\"After suspend/resume\"}"
    
    # Test password input during signal handling
    echo "Testing password input with signals..."
    echo "When prompted for password, enter a test password."
    echo "During password entry, a SIGWINCH will be sent to simulate terminal resize."
    
    # Start a background process to send SIGWINCH during password input
    (
        sleep 2
        echo "Sending SIGWINCH during password input..."
        kill -WINCH $MCPD_PID
    ) &
    
    send_request "password_prompt" "{}" 10
fi

echo "Tests completed successfully!"
echo "Trace file is available at: $TRACE_FILE"