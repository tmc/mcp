#!/bin/bash

# Local Authentication Demo for mcpd
echo "=== MCP Daemon Local Authentication Demo ==="
echo

# Create a simple echo server for testing
cat > echo_server.go << 'EOF'
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		
		// Simple echo response
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": fmt.Sprintf("Echo: %v", req),
					},
				},
			},
		}
		
		respBytes, _ := json.Marshal(resp)
		fmt.Println(string(respBytes))
	}
}
EOF

echo "Building test echo server..."
go build -o echo_server echo_server.go

echo
echo "=== Local Authentication Examples ==="
echo

echo "1. Basic local auth with command-line users:"
echo "./mcpd-local -enable-oauth -oauth-provider local -local-auth-users 'admin:password,user:secret' -http :8081 -- ./echo_server"
echo

echo "2. Local auth with users from file (users.txt):"
echo "# Create users.txt file:"
echo "admin:password123" > users.txt
echo "user:secret456" >> users.txt
echo "guest:welcome" >> users.txt
echo "# Run mcpd:"
echo "./mcpd-local -enable-oauth -oauth-provider local -local-auth-file users.txt -http :8081 -- ./echo_server"
echo

echo "3. Local auth with persistent JSON store:"
echo "./mcpd-local -enable-oauth -oauth-provider local -local-auth-users 'admin:admin' -local-auth-persist users.json -http :8081 -- ./echo_server"
echo

echo "4. Local auth with environment variables:"
echo "MCPD_USERS='dev:dev123,ops:ops456' ./mcpd-local -enable-oauth -oauth-provider local -http :8081 -- ./echo_server"
echo

echo "=== Testing Local Authentication ==="
echo

# Test 1: Basic command-line users
echo "Test 1: Starting mcpd with local auth and command-line users..."
timeout 10s ./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-users 'admin:admin,demo:demo' \
  -http :0 \
  -v \
  -- ./echo_server &

MCPD_PID=$!
sleep 2

# Find the actual port
HTTP_PORT=$(lsof -p $MCPD_PID -a -i tcp | grep LISTEN | awk '{print $9}' | cut -d: -f2)

if [ -n "$HTTP_PORT" ]; then
    echo "✅ Local auth server started on port $HTTP_PORT"
    echo "🌐 Access: http://localhost:$HTTP_PORT/login"
    echo "👤 Test users: admin/admin, demo/demo"
    sleep 3
else
    echo "❌ Failed to start server"
fi

# Clean up
kill $MCPD_PID 2>/dev/null
wait $MCPD_PID 2>/dev/null

echo
echo "Test 2: Testing with environment variables..."

# Test 2: Environment variables
MCPD_USERS='env_user:env_pass' timeout 10s ./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -http :0 \
  -v \
  -- ./echo_server &

MCPD_PID=$!
sleep 2

# Find the actual port
HTTP_PORT=$(lsof -p $MCPD_PID -a -i tcp | grep LISTEN | awk '{print $9}' | cut -d: -f2)

if [ -n "$HTTP_PORT" ]; then
    echo "✅ Environment variable auth server started on port $HTTP_PORT"
    echo "🌐 Access: http://localhost:$HTTP_PORT/login"
    echo "👤 Test user from env: env_user/env_pass"
    sleep 3
else
    echo "❌ Failed to start server with env users"
fi

# Clean up
kill $MCPD_PID 2>/dev/null
wait $MCPD_PID 2>/dev/null

echo
echo "Test 3: Testing file-based authentication..."

# Test 3: File-based auth
timeout 10s ./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-file users.txt \
  -http :0 \
  -v \
  -- ./echo_server &

MCPD_PID=$!
sleep 2

# Find the actual port
HTTP_PORT=$(lsof -p $MCPD_PID -a -i tcp | grep LISTEN | awk '{print $9}' | cut -d: -f2)

if [ -n "$HTTP_PORT" ]; then
    echo "✅ File-based auth server started on port $HTTP_PORT"
    echo "🌐 Access: http://localhost:$HTTP_PORT/login"
    echo "👤 Test users from file: admin/password123, user/secret456, guest/welcome"
    sleep 3
else
    echo "❌ Failed to start server with file auth"
fi

# Clean up
kill $MCPD_PID 2>/dev/null
wait $MCPD_PID 2>/dev/null

echo
echo "=== Local Authentication Features ==="
echo "✅ Username/password authentication (no third-party providers needed)"
echo "✅ Secure password hashing with bcrypt"
echo "✅ Session management with secure cookies"
echo "✅ Multiple user management options:"
echo "   - Command-line users (-local-auth-users)"
echo "   - File-based users (-local-auth-file)" 
echo "   - Environment variables (MCPD_USERS)"
echo "   - Persistent JSON store (-local-auth-persist)"
echo "✅ Built-in login/logout web interface"
echo "✅ Automatic default admin user creation"
echo "✅ Complete offline/air-gapped deployment support"
echo

echo "=== Security Notes ==="
echo "🔐 Passwords are hashed with bcrypt (not stored in plaintext)"
echo "🍪 Sessions use secure HTTP-only cookies"
echo "⏰ 24-hour session expiration"
echo "🔒 No external dependencies or third-party services"
echo "🏠 Perfect for local development and internal deployments"

# Clean up test files
rm -f echo_server echo_server.go users.txt users.json

echo
echo "Demo completed! ✨"