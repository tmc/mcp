#!/bin/bash
# Demo script for MCP9P namespace system

set -e

echo "MCP9P Demo - Plan9-inspired namespace for MCP"
echo "============================================"
echo

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Build tools if needed
if [ ! -d "bin" ]; then
    echo -e "${BLUE}Building tools...${NC}"
    make build
    echo
fi

# Start namespace server
echo -e "${GREEN}Starting namespace server...${NC}"
./bin/mcp-namespace -addr :9000 &
NS_PID=$!
sleep 2

# Register some services
echo -e "${GREEN}Registering services...${NC}"

# Local services
./bin/mcp-ns -s http://localhost:9000 -c register /services/echo \
    -type local \
    -transport stdio \
    -command echo \
    -args "Hello from echo service"

./bin/mcp-ns -s http://localhost:9000 -c register /services/time \
    -type local \
    -transport stdio \
    -command date

# Create namespace for AI services
./bin/mcp-ns -s http://localhost:9000 -c register /services/ai \
    -type namespace

# Remote services
./bin/mcp-ns -s http://localhost:9000 -c register /services/ai/gpt \
    -type remote \
    -transport http \
    -address "https://gpt.api/mcp" \
    -metadata "model=gpt-4,rate_limit=100"

./bin/mcp-ns -s http://localhost:9000 -c register /services/ai/claude \
    -type remote \
    -transport http \
    -address "https://claude.api/mcp" \
    -metadata "model=claude-3,rate_limit=50"

echo
echo -e "${GREEN}Listing all services:${NC}"
./bin/mcp-ns -s http://localhost:9000 -c list /services

echo
echo -e "${GREEN}Looking up echo service:${NC}"
./bin/mcp-ns -s http://localhost:9000 -c lookup /services/echo

echo
echo -e "${GREEN}Creating mount /local/echo -> /services/echo:${NC}"
./bin/mcp-mount -ns localhost:9000 /services/echo /local/echo

echo
echo -e "${GREEN}Creating bind /gpt -> /services/ai/gpt:${NC}"
./bin/mcp-mount -ns localhost:9000 -type bind /services/ai/gpt /gpt

echo
echo -e "${GREEN}Demo complete!${NC}"
echo "Namespace server running on PID $NS_PID"
echo "Press Ctrl+C to stop the server"

# Keep running until interrupted
wait $NS_PID