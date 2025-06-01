#!/bin/bash

set -e

echo "=== MCP Tool Graph Visualization Demo ==="
echo

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create output directory
OUTPUT_DIR="demo-output"
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR

echo -e "${BLUE}1. Building the tool...${NC}"
go build -o mcp-tool-graph ../cmd/mcp-tool-graph/main.go
echo -e "${GREEN}✓ Build complete${NC}"
echo

echo -e "${BLUE}2. Analyzing scripttest example...${NC}"
./mcp-tool-graph -target example/oauth_test.txt -output $OUTPUT_DIR/oauth -format react -verbose
echo -e "${GREEN}✓ Analysis complete${NC}"
echo

echo -e "${BLUE}3. Analyzing Go test example...${NC}"
./mcp-tool-graph -target example/integration_test.go -output $OUTPUT_DIR/integration -format react
echo -e "${GREEN}✓ Analysis complete${NC}"
echo

echo -e "${BLUE}4. Generating DOT format...${NC}"
./mcp-tool-graph -target example/oauth_test.txt -output $OUTPUT_DIR/oauth.dot -format dot
echo -e "${GREEN}✓ DOT file generated${NC}"
echo

echo -e "${BLUE}5. Starting web server...${NC}"
echo -e "${YELLOW}Visit http://localhost:8080 to view the OAuth test visualization${NC}"
echo -e "${YELLOW}Visit http://localhost:8081 to view the Go test visualization${NC}"
echo

# Start servers in background
echo "Starting servers..."
(cd $OUTPUT_DIR/oauth && python3 -m http.server 8080 > /dev/null 2>&1) &
PID1=$!
(cd $OUTPUT_DIR/integration && python3 -m http.server 8081 > /dev/null 2>&1) &
PID2=$!

echo "Servers running (PIDs: $PID1, $PID2)"
echo "Press Ctrl+C to stop"
echo

# Function to cleanup on exit
cleanup() {
    echo -e "\n${BLUE}Stopping servers...${NC}"
    kill $PID1 $PID2 2>/dev/null || true
    echo -e "${GREEN}Demo complete!${NC}"
}

trap cleanup EXIT

# Keep script running
while true; do
    sleep 1
done