#!/bin/bash
# Test mcp-probe in stdio mode with a simple echo simulation

# Create a simple mock server response
cat > mock_response.txt << 'EOF'
Content-Length: 198

{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-03-26","serverInfo":{"name":"mock-server","version":"1.0.0"},"capabilities":{"tools":{"listChanged":false}}}}
EOF

# Run mcp-probe in stdio mode with the mock response
./mcp-probe -v -timeout=2s < mock_response.txt