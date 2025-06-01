#!/bin/bash

echo "=== Simple Local Auth Test ==="

# Create a simple echo server
cat > test_server.go << 'EOF'
package main
import ("bufio"; "encoding/json"; "fmt"; "os")
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req map[string]interface{}
		json.Unmarshal(scanner.Bytes(), &req)
		resp := map[string]interface{}{
			"jsonrpc": "2.0", "id": req["id"],
			"result": map[string]interface{}{"content": []map[string]interface{}{{"type": "text", "text": "Local auth working!"}}},
		}
		respBytes, _ := json.Marshal(resp); fmt.Println(string(respBytes))
	}
}
EOF

go build -o test_server test_server.go

echo "Starting mcpd with local authentication..."
echo "Users: admin/admin, test/test"
echo "Access: http://localhost:8082/login"
echo "Press Ctrl+C to stop"
echo

./mcpd-local \
  -enable-oauth \
  -oauth-provider local \
  -local-auth-users 'admin:admin,test:test' \
  -http :8082 \
  -v \
  -- ./test_server

# Cleanup
rm -f test_server test_server.go