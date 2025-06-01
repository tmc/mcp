#!/bin/bash
# Demo script for mcp-trace-codegen

echo "=== MCP Trace to Go Code Generator Demo ==="
echo

# Create a sample trace file (or use existing one)
cat > sample-trace.mcp << 'EOF'
2024-01-15T10:00:00 -> initialize {"method":"initialize","params":{"protocolVersion":"1.0","capabilities":{},"clientInfo":{"name":"demo-client","version":"1.0"}}}
2024-01-15T10:00:01 <- initialize {"result":{"protocolVersion":"1.0","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"demo-server","version":"1.0"}}}
2024-01-15T10:00:02 -> tools/list {"method":"tools/list","params":{}}
2024-01-15T10:00:03 <- tools/list {"result":{"tools":[{"name":"get_weather","description":"Get current weather for a location","inputSchema":{"type":"object","properties":{"location":{"type":"string","description":"City name"},"units":{"type":"string","enum":["celsius","fahrenheit"]}},"required":["location"]}}]}}
2024-01-15T10:00:04 -> tools/call {"method":"tools/call","params":{"name":"get_weather","arguments":{"location":"San Francisco","units":"celsius"}}}
2024-01-15T10:00:05 <- tools/call {"result":{"content":[{"type":"text","text":"Current weather in San Francisco: 18°C, partly cloudy"}]}}
EOF

echo "Running real-time code generation from trace..."
echo "Press Ctrl+C to stop"
echo

# Run the generator with real-time display
cat sample-trace.mcp | go run . -realtime -package weather_server

# Or simulate real-time by adding lines one by one
echo
echo "=== Simulating real-time trace processing ==="
echo

# Clear the screen and run line-by-line
(
    while IFS= read -r line; do
        echo "$line"
        sleep 0.5  # Simulate real-time delay
    done < sample-trace.mcp
) | go run . -realtime -clear -package weather_server

echo
echo "Demo complete!"