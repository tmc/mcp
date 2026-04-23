#!/bin/bash
# Demo script for mcptrace-to-otel

set -e

echo "=== MCPTrace to OpenTelemetry Demo ==="
echo

# Build the tool
echo "Building mcptrace-to-otel..."
go build -o mcptrace-to-otel .

# Function to cleanup containers
cleanup() {
    echo -e "\nCleaning up Docker containers..."
    docker compose down
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Start the tracing infrastructure
echo -e "\nStarting tracing infrastructure..."
docker compose up -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Generate some example traces
echo -e "\nGenerating example MCP traces..."

# Basic trace
cat > basic.mcp << 'EOF'
# mcptrace:v1 traceparent=00-1234567890abcdef1234567890abcdef-1234567890abcdef-01
mcp-recv {"jsonrpc":"2.0","method":"initialize","id":1} # 1683000000.000 spanid=aaa1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}} # 1683000000.100 spanid=aaa2
mcp-recv {"jsonrpc":"2.0","method":"shutdown","id":2} # 1683000001.000 spanid=bbb1
mcp-send {"jsonrpc":"2.0","id":2,"result":null} # 1683000001.050 spanid=bbb2
EOF

# Shadow trace
cat > shadow.mcp << 'EOF'
# mcptrace:v1 traceparent=00-abcdef1234567890abcdef1234567890-abcdef1234567890-01 baggage=test=shadow-demo
mcp-recv {"jsonrpc":"2.0","method":"tools/call","params":{"name":"calc","args":{"expr":"2+2"}},"id":1} # 1683000000.000 spanid=ccc1
mcp-send {"jsonrpc":"2.0","id":1,"result":{"value":4}} # 1683000000.100 spanid=ccc2
# mcp-send {"jsonrpc":"2.0","id":1,"result":{"value":4,"cached":false}} # 1683000000.110 spanid=ddd2 linksto=ccc2 baggage=shadow=true
EOF

# Error trace
cat > error.mcp << 'EOF'
# mcptrace:v1
mcp-recv {"jsonrpc":"2.0","method":"invalid_method","id":1} # 1683000000.000 spanid=eee1
mcp-send {"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}} # 1683000000.050 spanid=eee2
EOF

echo -e "\nExporting traces to different backends..."

# Export to stdout
echo -e "\n1. Stdout export (debugging):"
./mcptrace-to-otel -f basic.mcp -type stdout | head -20

# Export to Jaeger
echo -e "\n2. Exporting to Jaeger..."
./mcptrace-to-otel -f basic.mcp -type jaeger -endpoint http://localhost:14268/api/traces
./mcptrace-to-otel -f shadow.mcp -type jaeger -endpoint http://localhost:14268/api/traces
./mcptrace-to-otel -f error.mcp -type jaeger -endpoint http://localhost:14268/api/traces
echo "Success! Traces sent to Jaeger."

# Export to Zipkin
echo -e "\n3. Exporting to Zipkin..."
./mcptrace-to-otel -f basic.mcp -type zipkin -endpoint http://localhost:9411/api/v2/spans
echo "Success! Traces sent to Zipkin."

# Export to OTLP
echo -e "\n4. Exporting to OTLP (Tempo)..."
./mcptrace-to-otel -f shadow.mcp -type otlp-grpc -endpoint localhost:4317
echo "Success! Traces sent to Tempo."

# Display URLs
echo -e "\n=== View your traces ==="
echo "Jaeger UI: http://localhost:16686"
echo "  - Search for service: 'mcp-trace'"
echo "  - Look for operations like 'mcp.recv.initialize'"
echo
echo "Zipkin UI: http://localhost:9411"
echo "  - Search for service: 'mcp-trace'"
echo
echo "Grafana: http://localhost:3000"
echo "  - Explore → Tempo → Search traces"
echo

echo -e "\n=== Demo Summary ==="
echo "1. Created three example trace files (basic.mcp, shadow.mcp, error.mcp)"
echo "2. Exported them to multiple backends (Jaeger, Zipkin, Tempo)"
echo "3. Shadow responses appear as events on the primary span"
echo "4. Error responses are marked with error status"
echo
echo "To stop all services: docker compose down"
echo "To view container logs: docker compose logs -f"