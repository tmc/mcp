#!/bin/bash
set -e

echo "Building mcpd..."
go build -o mcpd .

echo "Creating a simple MCP echo script..."
cat > echo.sh << 'EOF'
#!/bin/bash

# Simple MCP echo server that reads JSON-RPC requests and sends responses
while read line; do
  # Process initialize request
  if [[ "$line" == *"initialize"* ]]; then
    echo '{"jsonrpc":"2.0","id":1,"result":{"capabilities":{"server":true}}}'
  # Process echo request
  elif [[ "$line" == *"echo"* ]]; then
    echo '{"jsonrpc":"2.0","id":2,"result":{"message":"Hello from echo server!"}}'
  # Exit request
  elif [[ "$line" == *"exit"* ]]; then
    echo '{"jsonrpc":"2.0","id":3,"result":"goodbye"}'
    exit 0
  fi
done
EOF
chmod +x echo.sh

# Create a temporary trace file
TRACE_FILE=$(mktemp -t mcpd_test.XXXXXX.mcp)

echo "Starting mcpd with echo script..."
./mcpd -v -mode once -log-file "$TRACE_FILE" -- ./echo.sh &
MCPD_PID=$!

# Wait for mcpd to start and get the socket path
sleep 1
SOCKET_PATH=$(head -1 "$TRACE_FILE")
if [[ "$SOCKET_PATH" == "# mcptrace:v1" ]]; then
  # First line is the header, get the socket from stdout
  SOCKET_PATH=$(ps -p $MCPD_PID | grep mcpd | grep -o 'unix://[^ ]*' || echo "unix://unknown")
fi

echo "Socket path: $SOCKET_PATH"
SOCKET_FILE=${SOCKET_PATH#unix://}

echo "Sending initialize request..."
echo '{"jsonrpc":"2.0","method":"initialize","id":1}' | nc -U "$SOCKET_FILE"
sleep 1

echo "Sending echo request..."
echo '{"jsonrpc":"2.0","method":"echo","id":2,"params":{"message":"Hello"}}' | nc -U "$SOCKET_FILE"
sleep 1

echo "Sending exit request..."
echo '{"jsonrpc":"2.0","method":"exit","id":3}' | nc -U "$SOCKET_FILE"
sleep 1

echo "Stopping mcpd..."
kill $MCPD_PID

echo "Contents of trace file:"
cat "$TRACE_FILE"

# Check if trace file contains expected content
if grep -q "initialize" "$TRACE_FILE" && grep -q "echo" "$TRACE_FILE"; then
  echo -e "\033[32mTest passed! Trace file contains expected content.\033[0m"
else
  echo -e "\033[31mTest failed! Trace file missing expected content.\033[0m"
  exit 1
fi